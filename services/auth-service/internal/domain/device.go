package domain

import (
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uuid.UUID `gorm:"index;type:uuid"`
	DeviceID     string    `gorm:"size:64;index"` // ID от фронтенда
	DeviceName   string
	LastActiveAt time.Time
	CreatedAt    time.Time
}
