package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserCourse struct {
	UserID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	CourseID        string    `gorm:"primaryKey"` // ID из course-service
	Title           string    // Кешируем название
	CoverURL        string    // Кешируем обложку
	ProgressPercent int32     `gorm:"default:0"`
	Status          string    `gorm:"default:'active'"` // "active", "completed"
	LastAccessedAt  time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
