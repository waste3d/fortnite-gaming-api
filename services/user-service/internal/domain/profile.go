package domain

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email    string    `gorm:"uniqueIndex"`
	Username string
	AvatarID int `gorm:"default:1"`

	// === ПОЛЯ ПОДПИСКИ ===
	SubscriptionStatus string    `gorm:"default:'Обычный'"`
	CourseLimit        int       `gorm:"default:0"` // 0 = ничего нельзя
	DeviceLimit        int       `gorm:"default:1"` // мин. 1 устройство
	HasTgAccess        bool      `gorm:"default:false"`
	SubscriptionEndsAt time.Time // Дата окончания (2030 год по дефолту или null, если бессрочно)

	Streak       int       `gorm:"default:0"` // Текущая серия
	LastStreakAt time.Time // Когда последний раз серия была обновлена (или начата)

	CreatedAt time.Time
	UpdatedAt time.Time
}
