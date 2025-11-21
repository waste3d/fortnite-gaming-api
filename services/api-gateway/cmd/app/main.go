package main

import (
	"log"

	"gameplatform/services/api-gateway/internal/client"
	"gameplatform/services/api-gateway/internal/config"
	handlers "gameplatform/services/api-gateway/internal/transport/http"
)

func main() {
	// 1. Конфиг
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. gRPC Клиент для Auth
	authClient, err := client.NewAuthClient(cfg.AuthSvcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to Auth Service: %v", err)
	}
	log.Println("Connected to Auth Service at", cfg.AuthSvcUrl)

	// 3. Инициализация хендлеров
	authHandler := handlers.NewAuthHandler(authClient)

	// 4. Роутер
	router := handlers.NewRouter(authHandler)

	// 5. Запуск HTTP сервера
	log.Printf("API Gateway running on port %s", cfg.Port)
	if err := router.Run(cfg.Port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
