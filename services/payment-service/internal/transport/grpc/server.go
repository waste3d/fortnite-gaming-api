package grpc_server

import (
	"context"
	"fmt"
	"time"

	"payment-service/internal/repository"
	paymentpb "payment-service/pkg/paymentpb/proto/payment"

	userpb "github.com/waste3d/gameplatform-api/services/user-service/pkg/userpb/proto/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	repo       *repository.PaymentRepository
	userClient userpb.UserServiceClient
}

func NewPaymentServer(repo *repository.PaymentRepository, uc userpb.UserServiceClient) *PaymentServer {
	return &PaymentServer{repo: repo, userClient: uc}
}

func (s *PaymentServer) RedeemPromo(ctx context.Context, req *paymentpb.RedeemPromoRequest) (*paymentpb.RedeemPromoResponse, error) {
	// 1. Ищем код в БД
	promo, err := s.repo.GetPromoWithPlan(ctx, req.Code)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Промокод не найден")
	}

	// 2. Проверяем валидность
	if promo.ExpiresAt != nil && promo.ExpiresAt.Before(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "Срок действия промокода истек")
	}
	if promo.MaxUses > 0 && promo.UsedCount >= promo.MaxUses {
		return nil, status.Error(codes.ResourceExhausted, "Лимит использований этого кода исчерпан")
	}

	// 3. Вычисляем длительность
	plan := promo.Plan
	duration := plan.DefaultDurationDays
	if promo.OverrideDuration > 0 {
		duration = promo.OverrideDuration
	}

	// 4. Считаем дату окончания
	expiresAt := time.Now().Add(time.Duration(duration) * 24 * time.Hour)

	// 5. Вызываем User Service для обновления профиля
	_, err = s.userClient.SetSubscription(ctx, &userpb.SetSubscriptionRequest{
		UserId:      req.UserId,
		PlanName:    plan.Name,
		CourseLimit: int32(plan.CourseLimit),
		DeviceLimit: int32(plan.DeviceLimit),
		TgAccess:    plan.IsTgAccess,
		ExpiresAt:   expiresAt.Unix(),
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Ошибка активации подписки: %v", err)
	}

	// 6. Фиксируем использование кода
	_ = s.repo.IncrementUsage(ctx, promo.Code)

	return &paymentpb.RedeemPromoResponse{
		Success:   true,
		Message:   fmt.Sprintf("Подписка '%s' активирована на %d дней", plan.Name, duration),
		PlanName:  plan.Name,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (s *PaymentServer) GetPlans(ctx context.Context, req *paymentpb.GetPlansRequest) (*paymentpb.GetPlansResponse, error) {
	plans, _ := s.repo.GetAllPlans(ctx)
	var pbPlans []*paymentpb.Plan
	for _, p := range plans {
		pbPlans = append(pbPlans, &paymentpb.Plan{
			Id:          p.ID.String(),
			Name:        p.Name,
			Price:       int32(p.Price),
			Description: p.Description,
		})
	}
	return &paymentpb.GetPlansResponse{Plans: pbPlans}, nil
}
