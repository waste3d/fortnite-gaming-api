package repository

import (
	"auth-service/internal/domain"
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) Find(ctx context.Context, userID uuid.UUID, deviceID string) (*domain.Device, error) {
	var device domain.Device
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		First(&device).Error
	return &device, err
}

func (r *DeviceRepository) Count(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Device{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *DeviceRepository) Create(ctx context.Context, device *domain.Device) error {
	return r.db.WithContext(ctx).Create(device).Error
}

func (r *DeviceRepository) UpdateLastActive(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&domain.Device{}).
		Where("id = ?", id).
		Update("last_active_at", time.Now()).Error
}

func (r *DeviceRepository) List(ctx context.Context, userID uuid.UUID) ([]domain.Device, error) {
	var devices []domain.Device
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_active_at desc").
		Find(&devices).Error
	return devices, err
}

func (r *DeviceRepository) Delete(ctx context.Context, userID uuid.UUID, deviceID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Delete(&domain.Device{}).Error
}
