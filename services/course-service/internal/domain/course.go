package domain

import (
	"time"

	"github.com/google/uuid"
)

type Course struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title       string    `gorm:"index"`
	Description string
	Category    string `gorm:"index"`
	Duration    string
	CoverURL    string
	CloudLink   string

	// Связь один-ко-многим: У курса много уроков
	Lessons []Lesson `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE;"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Новая структура
type Lesson struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CourseID uuid.UUID `gorm:"type:uuid;index"`
	Title    string
	FileLink string // Ссылка на конкретный файл
	Order    int    // Для сортировки (1, 2, 3...)

	CreatedAt time.Time
}
