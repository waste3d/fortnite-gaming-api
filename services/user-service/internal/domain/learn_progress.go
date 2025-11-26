package domain

import (
	"time"

	"github.com/google/uuid"
)

type CompletedLesson struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;index"`
	CourseID  string    `gorm:"primaryKey;index"`
	LessonID  string    `gorm:"primaryKey"`
	CreatedAt time.Time
}
