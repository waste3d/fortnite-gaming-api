package application

import (
	"context"
	"gameplatform/services/auth-service/internal/domain"
	"gameplatform/services/auth-service/internal/infrastructure/cache"
	"gameplatform/services/auth-service/internal/infrastructure/repository"
	"gameplatform/services/auth-service/internal/infrastructure/security"
	"time"

	"github.com/google/uuid"
)

type AuthUseCase struct {
	userRepo     *repository.UserRepository
	tokenCache   *cache.TokenCache
	hasher       *security.PasswordHasher
	tokenManager *security.TokenManager
}

func NewAuthUseCase(
	ur *repository.UserRepository,
	tc *cache.TokenCache,
	h *security.PasswordHasher,
	tm *security.TokenManager,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:     ur,
		tokenCache:   tc,
		hasher:       h,
		tokenManager: tm,
	}
}

func (uc *AuthUseCase) Register(ctx context.Context, username, email, password string) (string, error) {
	hashPassword, err := uc.hasher.Hash(password)
	if err != nil {
		return "", err
	}

	user := &domain.User{
		ID:        uuid.New(),
		Username:  username,
		Email:     email,
		Password:  hashPassword,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return "", err
	}

	return user.ID.String(), nil
}
