package grpc_server

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"payment-service/internal/repository"
	paymentpb "payment-service/pkg/paymentpb/proto/payment"

	userpb "github.com/waste3d/gameplatform-api/services/user-service/pkg/userpb/proto/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Цены на дискретные товары
const (
	CoursePurchasePrice = 1000
	AvatarPurchasePrice = 250
)

type PaymentServer struct {
	paymentpb.UnimplementedPaymentServiceServer
	repo       *repository.PaymentRepository
	userClient userpb.UserServiceClient
}

func NewPaymentServer(repo *repository.PaymentRepository, uc userpb.UserServiceClient) *PaymentServer {
	return &PaymentServer{repo: repo, userClient: uc}
}

// НОВЫЙ МЕТОД ДЛЯ ПОКУПКИ
func (s *PaymentServer) PurchaseItem(ctx context.Context, req *paymentpb.PurchaseItemRequest) (*paymentpb.PurchaseItemResponse, error) {
	var price int
	var grantAction func() error

	switch req.ItemType {
	case "COURSE":
		price = CoursePurchasePrice
		grantAction = func() error {
			// Для покупки курса вызываем StartCourse в user-service
			_, err := s.userClient.StartCourse(ctx, &userpb.StartCourseRequest{
				UserId:   req.UserId,
				CourseId: req.ItemId,
				Title:    req.CourseTitle,
				CoverUrl: req.CourseCoverUrl,
			})
			return err
		}

	case "AVATAR":
		price = AvatarPurchasePrice
		avatarID, err := strconv.Atoi(req.ItemId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "неверный ID аватарки")
		}
		grantAction = func() error {
			// Для покупки аватарки вызываем UnlockAvatar
			_, err := s.userClient.UnlockAvatar(ctx, &userpb.UnlockAvatarRequest{
				UserId:   req.UserId,
				AvatarId: int32(avatarID),
			})
			return err
		}

	default:
		return nil, status.Error(codes.InvalidArgument, "неизвестный тип товара")
	}

	// 2. Списываем баланс
	_, err := s.userClient.ChangeBalance(ctx, &userpb.ChangeBalanceRequest{
		UserId: req.UserId,
		Amount: int32(-price),
	})
	if err != nil {
		return nil, status.Error(codes.ResourceExhausted, "Недостаточно снежинок")
	}

	// 3. Выдаем товар
	if err := grantAction(); err != nil {
		// Возвращаем деньги в случае ошибки
		_, _ = s.userClient.ChangeBalance(ctx, &userpb.ChangeBalanceRequest{
			UserId: req.UserId,
			Amount: int32(price),
		})
		return nil, status.Error(codes.Internal, "Не удалось выдать товар, средства возвращены")
	}

	return &paymentpb.PurchaseItemResponse{
		Success: true,
		Message: "Покупка успешно совершена!",
	}, nil
}

// Метод GetPlans теперь также возвращает цену в снежинках
func (s *PaymentServer) GetPlans(ctx context.Context, req *paymentpb.GetPlansRequest) (*paymentpb.GetPlansResponse, error) {
	plans, _ := s.repo.GetAllPlans(ctx)
	var pbPlans []*paymentpb.Plan
	for _, p := range plans {
		pbPlans = append(pbPlans, &paymentpb.Plan{
			Id:             p.ID.String(),
			Name:           p.Name,
			Price:          int32(p.Price),
			Description:    p.Description,
			SnowflakePrice: int32(p.SnowflakePrice), // <--- Добавили
		})
	}
	return &paymentpb.GetPlansResponse{Plans: pbPlans}, nil
}

// RedeemPromo остается без изменений...
func (s *PaymentServer) RedeemPromo(ctx context.Context, req *paymentpb.RedeemPromoRequest) (*paymentpb.RedeemPromoResponse, error) {
	// ... (весь ваш код для RedeemPromo остается здесь)
	codeClean := strings.ToUpper(strings.TrimSpace(req.Code))
	promo, err := s.repo.GetPromoWithPlan(ctx, codeClean)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Промокод не найден")
	}
	if promo.ExpiresAt != nil && promo.ExpiresAt.Before(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "Срок действия промокода истек")
	}
	if promo.MaxUses > 0 && promo.UsedCount >= promo.MaxUses {
		return nil, status.Error(codes.ResourceExhausted, "Лимит использований этого кода исчерпан")
	}
	used, err := s.repo.IsPromoActivatedByUser(ctx, req.UserId, codeClean)
	if err != nil {
		return nil, status.Error(codes.Internal, "Ошибка проверки промокода")
	}
	if used {
		return nil, status.Error(codes.AlreadyExists, "Вы уже активировали этот промокод")
	}
	plan := promo.Plan
	duration := plan.DefaultDurationDays
	if promo.OverrideDuration > 0 {
		duration = promo.OverrideDuration
	}
	expiresAt := time.Now().Add(time.Duration(duration) * 24 * time.Hour)
	if promo.Type == "ONE_COURSE" {
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
		_ = s.repo.IncrementUsage(ctx, promo.Code)
		_ = s.repo.SaveActivation(ctx, req.UserId, codeClean)
		return &paymentpb.RedeemPromoResponse{
			Success:  true,
			Message:  fmt.Sprintf("Активирован доступ к %d курсу(ам)!", slots),
			PlanName: "Бонус",
		}, nil
	}
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
	_ = s.repo.IncrementUsage(ctx, promo.Code)
	_ = s.repo.SaveActivation(ctx, req.UserId, codeClean)
	return &paymentpb.RedeemPromoResponse{
		Success:   true,
		Message:   fmt.Sprintf("Подписка '%s' активирована на %d дней", plan.Name, duration),
		PlanName:  plan.Name,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}
