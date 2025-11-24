package usecase

import (
	"context"
	"errors"
	"log"

	"auth-service/internal/domain"
	"auth-service/internal/infrastructure/cache"
	"auth-service/internal/infrastructure/email"
	"auth-service/internal/infrastructure/repository"
	"auth-service/internal/infrastructure/security"

	"github.com/google/uuid"
)

type AuthUseCase struct {
	userRepo     *repository.UserRepository
	tokenCache   *cache.TokenCache
	hasher       *security.PasswordHasher
	tokenManager *security.TokenManager
	emailSender  *email.EmailSender
}

func NewAuthUseCase(
	ur *repository.UserRepository,
	tc *cache.TokenCache,
	h *security.PasswordHasher,
	tm *security.TokenManager,
	es *email.EmailSender,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:     ur,
		tokenCache:   tc,
		hasher:       h,
		tokenManager: tm,
		emailSender:  es,
	}
}

func (uc *AuthUseCase) Register(ctx context.Context, username, email, password string) (string, error) {
	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return "", err
	}

	user := &domain.User{
		ID:       uuid.New(),
		Username: username,
		Email:    email,
		Password: hash,
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
		return "", "", err
	}

	cachedID, err := uc.tokenCache.CheckRefresh(ctx, oldRefreshToken)
	if err != nil || cachedID != userID {
		return "", "", errors.New("token revoked")
	}
	// Удаляем старый
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

	if err := uc.tokenCache.SaveRefresh(ctx, userID, refresh); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (uc *AuthUseCase) ForgotPassword(ctx context.Context, email string) error {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// ВАЖНО: С точки зрения безопасности, мы не должны говорить, что email не найден.
		return err
	}

	resetToken := uuid.New().String()

	if err := uc.tokenCache.SaveResetToken(ctx, resetToken, user.ID.String()); err != nil {
		return err
	}

	log.Printf("Attempting to send email to: %s", user.Email) // <-- ЛОГ 1

	go func() {
		err := uc.emailSender.SendResetEmail(user.Email, resetToken)
		if err != nil {
			// Этот лог покажет точную причину ошибки SMTP
			log.Printf("ERROR: Failed to send email to %s: %v", user.Email, err)
		} else {
			log.Printf("SUCCESS: Email sent to %s", user.Email)
		}
	}()

	return nil
}

func (uc *AuthUseCase) ResetPassword(ctx context.Context, token, newPassword string) error {
	userIDStr, err := uc.tokenCache.GetResetToken(ctx, token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	userID, _ := uuid.Parse(userIDStr)

	hash, err := uc.hasher.Hash(newPassword)
	if err != nil {
		return err
	}

	if err := uc.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}

	// Удаляем токен, чтобы его нельзя было использовать повторно
	_ = uc.tokenCache.DeleteResetToken(ctx, token)

	return nil
}
