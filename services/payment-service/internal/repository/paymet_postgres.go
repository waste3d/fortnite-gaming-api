package repository

import (
	"context"
	"payment-service/internal/domain"

	"gorm.io/gorm"
)

type PaymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Поиск промокода с жадной загрузкой плана
func (r *PaymentRepository) GetPromoWithPlan(ctx context.Context, code string) (*domain.PromoCode, error) {
	var promo domain.PromoCode
	// ILIKE делает поиск нечувствительным к регистру (Start3 == start3)
	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("code ILIKE ?", code).
		First(&promo).Error
	return &promo, err
}

// Увеличение счетчика использований
func (r *PaymentRepository) IncrementUsage(ctx context.Context, code string) error {
	return r.db.WithContext(ctx).Model(&domain.PromoCode{}).
		Where("code = ?", code).
		Update("used_count", gorm.Expr("used_count + 1")).Error
}

// Получить все планы
func (r *PaymentRepository) GetAllPlans(ctx context.Context) ([]domain.Plan, error) {
	var plans []domain.Plan
	err := r.db.WithContext(ctx).Order("price asc").Find(&plans).Error
	return plans, err
}
