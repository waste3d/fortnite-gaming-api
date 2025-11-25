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

	CreatedAt time.Time
	UpdatedAt time.Time
}
