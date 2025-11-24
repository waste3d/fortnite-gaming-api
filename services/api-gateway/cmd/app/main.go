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

	// 3. Инициализация хендлеров
	authHandler := handlers.NewAuthHandler(authClient)

	// 4. Роутер
	router := handlers.NewRouter(authHandler, rateLimiter)

	// 5. Запуск HTTP сервера
	log.Printf("API Gateway running on port %s", cfg.Port)
	if err := router.Run(cfg.Port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
