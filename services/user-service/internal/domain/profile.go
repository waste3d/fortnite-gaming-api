package domain

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email              string    `gorm:"uniqueIndex"`
	Username           string
	AvatarID           int    `gorm:"default:1"`
	SubscriptionStatus string `gorm:"default:'Обычный'"` // "Обычный", "Базовый", "Профессиональный", "Пробный", "Премиум"

	CreatedAt time.Time
	UpdatedAt time.Time
}
