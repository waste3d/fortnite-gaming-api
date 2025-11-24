package repository

import (
	"context"
	"user-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProfileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) Create(ctx context.Context, profile *domain.Profile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *ProfileRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	var profile domain.Profile
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&profile).Error
	return &profile, err
}

func (r *ProfileRepository) Update(ctx context.Context, profile *domain.Profile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

func (r *ProfileRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
	return r.db.WithContext(ctx).Model(&domain.Profile{}).
		Where("id = ?", id).
		Update("email", email).Error
}

func (r *ProfileRepository) UpdateUsername(ctx context.Context, id uuid.UUID, username string) error {
	result := r.db.WithContext(ctx).Model(&domain.Profile{}).
		Where("id = ?", id).
		Update("username", username)

	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Обновляем только AvatarID
func (r *ProfileRepository) UpdateAvatar(ctx context.Context, id uuid.UUID, avatarID int) error {
	return r.db.WithContext(ctx).Model(&domain.Profile{}).
		Where("id = ?", id).
		Update("avatar_id", avatarID).Error
}
