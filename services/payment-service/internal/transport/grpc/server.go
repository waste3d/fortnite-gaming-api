package grpc_server

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
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
	// Нормализуем код (все буквы большие), чтобы "Start3" и "START3" считались одним кодом
	codeClean := strings.ToUpper(strings.TrimSpace(req.Code))

	// 1. Ищем код в БД
	promo, err := s.repo.GetPromoWithPlan(ctx, codeClean)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Промокод не найден")
	}

	// 2. Проверяем валидность (сроки и лимиты)
	if promo.ExpiresAt != nil && promo.ExpiresAt.Before(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "Срок действия промокода истек")
	}
	if promo.MaxUses > 0 && promo.UsedCount >= promo.MaxUses {
		return nil, status.Error(codes.ResourceExhausted, "Лимит использований этого кода исчерпан")
	}

	// === 3. НОВАЯ ПРОВЕРКА: Активировал ли юзер этот код раньше? ===
	used, err := s.repo.IsPromoActivatedByUser(ctx, req.UserId, codeClean)
	if err != nil {
		return nil, status.Error(codes.Internal, "Ошибка проверки промокода")
	}
	if used {
		return nil, status.Error(codes.AlreadyExists, "Вы уже активировали этот промокод")
	}

	// 4. Вычисляем длительность
	plan := promo.Plan
	duration := plan.DefaultDurationDays
	if promo.OverrideDuration > 0 {
		duration = promo.OverrideDuration
	}

	// 5. Считаем дату окончания
	expiresAt := time.Now().Add(time.Duration(duration) * 24 * time.Hour)

	if promo.Type == "ONE_COURSE" {
		// Начисляем слоты
		slots := promo.ValueInt
		if slots == 0 {
			slots = 1
		}

		_, err := s.userClient.AddCourseLimit(ctx, &userpb.AddCourseLimitRequest{
			UserId: req.UserId,
			Count:  int32(slots),
		})
		if err != nil {
			return nil, status.Error(codes.Internal, "Ошибка начисления слота")
		}

		// Фиксируем использование
		_ = s.repo.IncrementUsage(ctx, promo.Code)
		_ = s.repo.SaveActivation(ctx, req.UserId, codeClean)

		return &paymentpb.RedeemPromoResponse{
			Success:  true,
			Message:  fmt.Sprintf("Активирован доступ к %d курсу(ам)!", slots),
			PlanName: "Бонус",
		}, nil
	}

	// 6. Вызываем User Service
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

	// 7. Фиксируем использование кода
	// а) Увеличиваем глобальный счетчик
	_ = s.repo.IncrementUsage(ctx, promo.Code)
	// б) Записываем, что ЭТОТ юзер активировал ЭТОТ код (чтобы не смог второй раз)
	_ = s.repo.SaveActivation(ctx, req.UserId, codeClean)

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

func (s *PaymentServer) SpinWheel(ctx context.Context, req *paymentpb.SpinWheelRequest) (*paymentpb.SpinWheelResponse, error) {
	const Cost = 100 // Цена 100 снежинок

	// 1. Списываем баланс
	res, err := s.userClient.ChangeBalance(ctx, &userpb.ChangeBalanceRequest{
		UserId: req.UserId,
		Amount: -Cost,
	})
	if err != nil || !res.Success {
		return nil, status.Error(codes.ResourceExhausted, "Недостаточно снежинок (нужно 100)")
	}

	// 2. Логика рандома
	// 0-5: Эксклюзивная аватарка (5%)
	// 5-15: +1 слот для курса (10%)
	// 15-45: +50 снежинок (возврат части) (30%)
	// 45-100: Пусто / Утешительный приз (55%)

	rand.Seed(time.Now().UnixNano())
	chance := rand.Intn(100)

	var pType string
	var pValue int32
	var msg string

	if chance < 5 {
		// Редкая аватарка (ID от 10 до 20, например)
		avatarID := int32(rand.Intn(10) + 10) // 10..19
		pType = "AVATAR"
		pValue = avatarID
		msg = "ЭКСКЛЮЗИВНАЯ АВАТАРКА!"
		s.userClient.UnlockAvatar(ctx, &userpb.UnlockAvatarRequest{UserId: req.UserId, AvatarId: avatarID})

	} else if chance < 15 {
		pType = "SLOT"
		pValue = 1
		msg = "+1 СЛОТ ДЛЯ КУРСА!"
		s.userClient.AddCourseLimit(ctx, &userpb.AddCourseLimitRequest{UserId: req.UserId, Count: 1})

	} else if chance < 45 {
		pType = "BALANCE"
		pValue = 50
		msg = "+50 СНЕЖИНОК"
		s.userClient.ChangeBalance(ctx, &userpb.ChangeBalanceRequest{UserId: req.UserId, Amount: 50})

	} else {
		pType = "EMPTY"
		pValue = 0
		msg = "Эх, пусто... Попробуй еще!"
	}

	return &paymentpb.SpinWheelResponse{
		PrizeType:        pType,
		PrizeValue:       pValue,
		Message:          msg,
		RemainingBalance: res.NewBalance, // Возвращаем остаток (не учитывая приз, если это был баланс, но для UI сойдет)
	}, nil
}
