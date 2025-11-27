package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/redis/go-redis/v9"
	"github.com/waste3d/gameplatform-api/services/course-service/config"
	"github.com/waste3d/gameplatform-api/services/course-service/internal/domain"
	"github.com/waste3d/gameplatform-api/services/course-service/internal/infrastructure/repository"
	grpc_server "github.com/waste3d/gameplatform-api/services/course-service/internal/transport/grpc"
	coursepb "github.com/waste3d/gameplatform-api/services/course-service/pkg/coursepb/proto/course"
	userpb "github.com/waste3d/gameplatform-api/services/user-service/pkg/userpb/proto/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Config load failed: %v", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("DB connect failed: %v", err)
	}

	// Миграции
	db.AutoMigrate(&domain.Course{}, &domain.Lesson{})

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Подключение к User Service (для проверки прав)
	userConn, err := grpc.NewClient(cfg.UserSvcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("User Service connect failed: %v", err)
	}
	userClient := userpb.NewUserServiceClient(userConn)

	// Запуск сервера
	repo := repository.NewCourseRepository(db, rdb)
	courseServer := grpc_server.NewCourseServer(repo, userClient)

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}

	s := grpc.NewServer()
	coursepb.RegisterCourseServiceServer(s, courseServer)

	log.Printf("Course Service running on %s", cfg.GRPCPort)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Serve failed: %v", err)
	}
}
