package repository

import (
	"context"
	"errors"
	"time"

	"auth-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GORM Модель
type UserGorm struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Username  string    `gorm:"uniqueIndex;not null;size:50"`
	Email     string    `gorm:"uniqueIndex;not null;size:100"`
	Password  string    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserGorm) TableName() string {
	return "users"
}

func (ug *UserGorm) ToDomain() *domain.User {
	return &domain.User{
		ID:        ug.ID,
		Username:  ug.Username,
		Email:     ug.Email,
		Password:  ug.Password,
		CreatedAt: ug.CreatedAt,
		UpdatedAt: ug.UpdatedAt,
	}
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	gormUser := &UserGorm{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Password: user.Password,
	}

	result := r.db.WithContext(ctx).Create(gormUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrUserAlreadyExists
		}
		return result.Error
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var userModel UserGorm
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&userModel).Error
	if err != nil {
		return nil, err // GORM вернет ErrRecordNotFound, обработаем в UseCase
	}
	return userModel.ToDomain(), nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, newPassword string) error {

	return r.db.WithContext(ctx).Model(&UserGorm{}).
		Where("id = ?", userID).
		Update("password", newPassword).Error
}

func (r *UserRepository) UpdateEmail(ctx context.Context, userID uuid.UUID, newEmail string) error {
	return r.db.WithContext(ctx).Model(&UserGorm{}).
		Where("id = ?", userID).
		Update("email", newEmail).Error
}
