package main

import (
	"context"
	"fmt"
	"log"
	"net"

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

	// === SEED (Наполнение данными, если пусто) ===
	var count int64
	db.Model(&domain.Course{}).Count(&count)
	if count == 0 {
		repo := repository.NewCourseRepository(db)
		// Пример 1
		repo.Create(context.Background(), &domain.Course{
			Title:       "Fullstack Python Разработчик",
			Description: "Полный курс по разработке веб-приложений на Python + Django + Vue.js. От основ до деплоя.",
			Category:    "Программирование",
			Duration:    "45 часов",
			CoverURL:    "https://images.unsplash.com/photo-1526379095098-d400fd0bf935?auto=format&fit=crop&w=800&q=80",
			CloudLink:   "https://cloud.mail.ru/public/test/python",
		})
		// Пример 2
		repo.Create(context.Background(), &domain.Course{
			Title:       "UX/UI Дизайн с нуля",
			Description: "Научитесь создавать удобные и красивые интерфейсы в Figma.",
			Category:    "Дизайн",
			Duration:    "20 часов",
			CoverURL:    "https://images.unsplash.com/photo-1561070791-2526d30994b5?auto=format&fit=crop&w=800&q=80",
			CloudLink:   "https://cloud.mail.ru/public/test/design",
		})
		log.Println(">>> DB Seeded with default courses")
	}

	// Подключение к User Service (для проверки прав)
	userConn, err := grpc.NewClient(cfg.UserSvcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("User Service connect failed: %v", err)
	}
	userClient := userpb.NewUserServiceClient(userConn)

	// Запуск сервера
	repo := repository.NewCourseRepository(db)
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
