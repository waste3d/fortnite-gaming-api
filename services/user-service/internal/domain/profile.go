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

	SubscriptionStatus string `gorm:"default:'Обычный'"`
	CourseLimit        int    `gorm:"default:0"`
	DeviceLimit        int    `gorm:"default:1"`
	HasTgAccess        bool   `gorm:"default:false"`
	SubscriptionEndsAt time.Time

	Streak       int `gorm:"default:0"`
	LastStreakAt time.Time

	// === НОВЫЕ ПОЛЯ ===
	Balance        int `gorm:"default:0"` // Снежинки
	CompletedCount int `gorm:"default:0"` // Статистика для лидерборда

	// Связь с открытыми аватарками
	UnlockedAvatars []UnlockedAvatar `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Таблица для хранения купленных/выигранных аватарок
type UnlockedAvatar struct {
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey;index"`
	AvatarID int       `gorm:"primaryKey"` // ID пресета (1-20)
}
