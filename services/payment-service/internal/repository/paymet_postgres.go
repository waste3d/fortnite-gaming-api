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
	// Preload("Plan") сработает корректно даже с nil, если настроен правильно,
	// но лучше добавить условие или обработку ошибок. GORM обычно справляется.
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

// Проверка: активировал ли этот юзер этот код ранее?
func (r *PaymentRepository) IsPromoActivatedByUser(ctx context.Context, userID, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.PromoActivation{}).
		Where("user_id = ? AND code = ?", userID, code).
		Count(&count).Error
	return count > 0, err
}

// Записать факт активации
func (r *PaymentRepository) SaveActivation(ctx context.Context, userID, code string) error {
	return r.db.WithContext(ctx).Create(&domain.PromoActivation{
		UserID: userID,
		Code:   code,
	}).Error
}
