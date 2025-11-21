package main

import (
	"context"
	"fmt"
	"gameplatform/services/auth-service/internal/application"
	"gameplatform/services/auth-service/internal/config"
	"gameplatform/services/auth-service/internal/domain"
	"gameplatform/services/auth-service/internal/infrastructure/cache"
	"gameplatform/services/auth-service/internal/infrastructure/repository"
	"gameplatform/services/auth-service/internal/infrastructure/security"
	grpc_handler "gameplatform/services/auth-service/internal/transport/grpc"
	authpb "gameplatform/services/auth-service/pkg/authpb/proto/auth"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		log.Fatalf("Failed to migrate DB: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	tokenCache := cache.NewTokenCache(rdb)
	hasher := security.NewPasswordHasher()
	tokenManager := security.NewTokenManager(config.AccessSecret, config.RefreshSecret)

	authUseCase := application.NewAuthUseCase(userRepo, tokenCache, hasher, tokenManager)
	authServer := grpc_handler.NewAuthServer(authUseCase)

	lis, err := net.Listen("tcp", config.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcServer, authServer)

	reflection.Register(grpcServer)

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
