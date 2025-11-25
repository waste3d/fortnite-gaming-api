package main

import (
	authpb "auth-service/pkg/authpb/proto/auth"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"auth-service/config"
	"auth-service/internal/application/usecase"
	"auth-service/internal/domain"
	"auth-service/internal/infrastructure/cache"
	"auth-service/internal/infrastructure/email"
	"auth-service/internal/infrastructure/repository"
	"auth-service/internal/infrastructure/security"
	grpc_server "auth-service/internal/transport/grpc"
	userpb "auth-service/pkg/userpb/proto/user"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {

	config, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.DBHost, config.DBUser, config.DBPassword, config.DBName, config.DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.Device{}); err != nil {
		log.Fatalf("Failed to migrate DB: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	userConn, err := grpc.NewClient(config.UserSvcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to User Service: %v", err)
	}
	defer userConn.Close()

	userClient := userpb.NewUserServiceClient(userConn)

	userRepo := repository.NewUserRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	tokenCache := cache.NewTokenCache(rdb)
	hasher := security.NewPasswordHasher()
	tokenManager := security.NewTokenManager(config.AccessSecret, config.RefreshSecret)
	emailSender := email.NewEmailSender(config.APIKey, config.SMTPEmail, config.FrontendURL)
	authUseCase := usecase.NewAuthUseCase(userRepo, tokenCache, hasher, tokenManager, emailSender, userClient, deviceRepo)
	authServer := grpc_server.NewAuthServer(authUseCase)

	lis, err := net.Listen("tcp", config.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, authServer)

	log.Printf("Auth Service is running on port %s...", config.GRPCPort)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("Shutting down server...")
	grpcServer.GracefulStop()
}
