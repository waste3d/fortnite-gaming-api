package main

import (
	"context"
	"log"

	"api-gateway/internal/client"
	"api-gateway/internal/config"
	"api-gateway/internal/middleware"
	handlers "api-gateway/internal/transport/http"

	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. Конфиг
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.REDIS_ADDR,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis at", cfg.REDIS_ADDR)

	rateLimiter := middleware.NewRateLimiter(rdb)

	// 2. gRPC Клиент для Auth
	authClient, err := client.NewAuthClient(cfg.AuthSvcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to Auth Service: %v", err)
	}
	log.Println("Connected to Auth Service at", cfg.AuthSvcUrl)

	// 3. gRPC Клиент для User
	userClient, err := client.NewUserClient(cfg.UserSvcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to User Service: %v", err)
	}
	log.Println("Connected to User Service at", cfg.UserSvcUrl)

	// 4. gRPC Клиент для Course
	courseClient, err := client.NewCourseClient(cfg.CourseSvcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to Course Service: %v", err)
	}
	log.Println("Connected to Course Service at", cfg.CourseSvcUrl)

	paymentClient, err := client.NewPaymentClient(cfg.PaymentSvcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to Payment Service: %v", err)
	}
	log.Println("Connected to Payment Service at", cfg.PaymentSvcUrl)

	// 3. Инициализация хендлеров
	authHandler := handlers.NewAuthHandler(authClient)
	userHandler := handlers.NewUserHandler(userClient, authClient)
	courseHandler := handlers.NewCourseHandler(courseClient, userClient)
	paymentHandler := handlers.NewPaymentHandler(paymentClient)
	// 4. Роутер
	router := handlers.NewRouter(authHandler, userHandler, rateLimiter, authClient, courseHandler, paymentHandler)

	// 5. Запуск HTTP сервера
	log.Printf("API Gateway running on port %s", cfg.Port)
	if err := router.Run(cfg.Port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
