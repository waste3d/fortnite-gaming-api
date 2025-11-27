package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/redis/go-redis/v9"
	"github.com/waste3d/gameplatform-api/services/user-service/config"
	"github.com/waste3d/gameplatform-api/services/user-service/internal/domain"
	"github.com/waste3d/gameplatform-api/services/user-service/internal/infrastructure/repository"
	grpc_server "github.com/waste3d/gameplatform-api/services/user-service/internal/transport/grpc"
	userpb "github.com/waste3d/gameplatform-api/services/user-service/pkg/userpb/proto/user"

	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. Загрузка конфига
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Подключение к БД
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// 3. Миграции (создаст таблицу profiles)
	log.Println("Running migrations...")
	if err := db.AutoMigrate(&domain.Profile{}, &domain.UserCourse{}, &domain.CompletedLesson{}); err != nil {
		log.Fatalf("Failed to migrate DB: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// 4. Инициализация слоев
	profileRepo := repository.NewProfileRepository(db, rdb)
	userServer := grpc_server.NewUserServer(profileRepo)

	// 5. Запуск gRPC сервера
	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, userServer)

	log.Printf("User Service running on %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
