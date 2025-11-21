package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type User struct {
	ID        uuid.UUID
	Email     string
	Username  string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
