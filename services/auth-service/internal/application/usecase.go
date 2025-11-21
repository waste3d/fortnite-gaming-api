package application

import (
	"context"
	"errors"
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

func (uc *AuthUseCase) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", errors.New("invalid credentials")
	}

	if err := uc.hasher.Compare(user.Password, password); err != nil {
		return "", "", errors.New("invalid credentials")
	}

	return uc.generateAndSaveTokens(ctx, user.ID.String())
}

func (uc *AuthUseCase) Refresh(ctx context.Context, oldRefreshToken string) (string, string, error) {
	userID, err := uc.tokenManager.ValidateRefreshToken(oldRefreshToken)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}
	cachedUserID, err := uc.tokenCache.CheckRefresh(ctx, oldRefreshToken)
	if err != nil || cachedUserID != userID {
		return "", "", errors.New("refresh token revoked or expired")
	}

	_ = uc.tokenCache.DeleteRefresh(ctx, oldRefreshToken)

	return uc.generateAndSaveTokens(ctx, userID)
}

func (uc *AuthUseCase) Logout(ctx context.Context, refreshToken string) error {
	return uc.tokenCache.DeleteRefresh(ctx, refreshToken)
}

func (uc *AuthUseCase) ValidateAccess(token string) (string, error) {
	return uc.tokenManager.ValidateAccessToken(token)
}

func (uc *AuthUseCase) generateAndSaveTokens(ctx context.Context, userID string) (string, string, error) {
	access, refresh, err := uc.tokenManager.Generate(userID)
	if err != nil {
		return "", "", err
	}

	// Сохраняем refresh в Redis
	if err := uc.tokenCache.SaveRefresh(ctx, userID, refresh); err != nil {
		return "", "", err
	}

	return access, refresh, nil
}
