package main

import (
	"fmt"
	"log"
	"net"

	"payment-service/config"
	"payment-service/internal/domain"
	"payment-service/internal/repository"
	grpc_server "payment-service/internal/transport/grpc"
	paymentpb "payment-service/pkg/paymentpb/proto/payment"

	userpb "github.com/waste3d/gameplatform-api/services/user-service/pkg/userpb/proto/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig(".") // Предполагаем наличие загрузчика конфига
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("DB Connection failed:", err)
	}

	// Миграция
	db.AutoMigrate(&domain.Plan{}, &domain.PromoCode{}, &domain.PromoActivation{})

	// Подключение к User Service
	userConn, err := grpc.NewClient(cfg.UserSvcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to connect to User Service:", err)
	}
	userClient := userpb.NewUserServiceClient(userConn)

	repo := repository.NewPaymentRepository(db)
	srv := grpc_server.NewPaymentServer(repo, userClient)

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}

	grpcServer := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(grpcServer, srv)

	log.Printf("Payment Service running on %s", cfg.GRPCPort)
	grpcServer.Serve(lis)
}
