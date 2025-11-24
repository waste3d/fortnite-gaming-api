package main

import (
	"fmt"
	"log"
	"net"

	"user-service/config"
	"user-service/internal/domain"
	"user-service/internal/infrastructure/repository"
	grpc_server "user-service/internal/transport/grpc"
	userpb "user-service/pkg/userpb/proto/user"

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
	if err := db.AutoMigrate(&domain.Profile{}); err != nil {
		log.Fatalf("Failed to migrate DB: %v", err)
	}

	// 4. Инициализация слоев
	profileRepo := repository.NewProfileRepository(db)
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
