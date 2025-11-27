package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"auth-service/internal/domain"
	"auth-service/internal/infrastructure/cache"
	"auth-service/internal/infrastructure/email"
	"auth-service/internal/infrastructure/repository"
	"auth-service/internal/infrastructure/security"
	userpb "auth-service/pkg/userpb/proto/user"

	"github.com/google/uuid"
)

type AuthUseCase struct {
	userRepo     *repository.UserRepository
	tokenCache   *cache.TokenCache
	hasher       *security.PasswordHasher
	tokenManager *security.TokenManager
	emailSender  *email.EmailSender
	userClient   userpb.UserServiceClient
	deviceRepo   *repository.DeviceRepository
}

func NewAuthUseCase(
	ur *repository.UserRepository,
	tc *cache.TokenCache,
	h *security.PasswordHasher,
	tm *security.TokenManager,
	es *email.EmailSender,
	uc userpb.UserServiceClient,
	dr *repository.DeviceRepository,
) *AuthUseCase {
	return &AuthUseCase{
		userRepo:     ur,
		tokenCache:   tc,
		hasher:       h,
		tokenManager: tm,
		emailSender:  es,
		userClient:   uc,
		deviceRepo:   dr,
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

	_, err = uc.userClient.CreateProfile(ctx, &userpb.CreateProfileRequest{
		UserId:   user.ID.String(),
		Email:    email,
		Username: username,
	})
	if err != nil {
		// Логируем ошибку, но юзера в Auth создали.
		// В идеале: удалять юзера из Auth (Rollback)
		log.Printf("Error creating profile: %v", err)
	}

	return user.ID.String(), nil
}

func (uc *AuthUseCase) Login(ctx context.Context, email, password, deviceID, deviceName string) (string, string, error) {
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", errors.New("invalid credentials")
	}
	if err := uc.hasher.Compare(user.Password, password); err != nil {
		return "", "", errors.New("invalid credentials")
	}

	profile, err := uc.userClient.GetProfile(ctx, &userpb.GetProfileRequest{
		UserId: user.ID.String(),
	})
	if err != nil {
		return "", "", errors.New("failed to get profile")
	}

	deviceLimit := int(profile.DeviceLimit)

	// Если подписка истекла (и это не админ), сбрасываем лимит до 1
	// (Хотя User Service при SetSubscription уже может это регулировать, но тут доп. защита)
	if profile.SubscriptionStatus != "admin" && profile.SubscriptionStatus != "Обычный" {
		// Если expires_at > 0 и текущее время > expires_at
		if profile.ExpiresAt > 0 && time.Now().Unix() > profile.ExpiresAt {
			deviceLimit = 1
		}
	}

	// Передаем ЧИСЛО (limit), а не название статуса
	err = uc.checkDeviceLimit(ctx, user.ID, deviceID, deviceName, deviceLimit)
	if err != nil {
		return "", "", err
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

func (uc *AuthUseCase) Logout(ctx context.Context, refreshToken string, deviceID string) error {
	log.Printf("DEBUG: Logout called. DeviceID: '%s'", deviceID)
	// 1. Пытаемся получить UserID из токена, чтобы удалить устройство
	userIDStr, err := uc.tokenManager.ValidateRefreshToken(refreshToken)

	// Если токен валиден и передан DeviceID, удаляем устройство из БД
	if err == nil && userIDStr != "" && deviceID != "" {
		if uid, errParse := uuid.Parse(userIDStr); errParse == nil {
			_ = uc.deviceRepo.Delete(ctx, uid, deviceID)
		}
	}

	// 2. Всегда удаляем сам токен из Redis (разлогин)
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

func (uc *AuthUseCase) RequestEmailChange(ctx context.Context, userID, newEmail string) error {
	// Проверка на занятость
	exists, _ := uc.userRepo.GetByEmail(ctx, newEmail)
	if exists != nil {
		return errors.New("email already taken")
	}

	token := uuid.New().String()
	if err := uc.tokenCache.SaveEmailChangeToken(ctx, token, userID, newEmail); err != nil {
		return err
	}

	go uc.emailSender.SendEmailChangeConfirmation(newEmail, token)
	return nil
}

func (uc *AuthUseCase) ConfirmEmailChange(ctx context.Context, token string) error {
	userIDStr, newEmail, err := uc.tokenCache.GetEmailChangeData(ctx, token)
	if err != nil {
		return errors.New("invalid token")
	}
	uid, _ := uuid.Parse(userIDStr)

	// Обновляем в Auth DB
	if err := uc.userRepo.UpdateEmail(ctx, uid, newEmail); err != nil {
		return err
	}

	// Обновляем в User DB (Sync)
	_, err = uc.userClient.SyncEmail(ctx, &userpb.SyncEmailRequest{
		UserId:   userIDStr,
		NewEmail: newEmail,
	})
	if err != nil {
		log.Printf("Failed to sync email with User Service: %v", err)
		// Тут можно вернуть ошибку, но Email в Auth уже изменен.
	}

	uc.tokenCache.DeleteEmailChangeToken(ctx, token)
	return nil
}

// Вспомогательная функция проверки
func (uc *AuthUseCase) checkDeviceLimit(ctx context.Context, userID uuid.UUID, deviceID, deviceName string, limit int) error {
	// Проверяем, есть ли устройство
	existingDevice, err := uc.deviceRepo.Find(ctx, userID, deviceID)

	if err == nil {
		return uc.deviceRepo.UpdateLastActive(ctx, existingDevice.ID)
	}

	// Устройства нет. Проверяем переданный лимит.
	// (Удалите чтение config.SubscriptionLimits отсюда!)

	currentCount, err := uc.deviceRepo.Count(ctx, userID)
	if err != nil {
		return err
	}

	if currentCount >= int64(limit) {
		return fmt.Errorf("device limit (%d) reached for your subscription", limit)
	}

	// Лимит не превышен — регистрируем
	newDevice := &domain.Device{
		UserID:       userID,
		DeviceID:     deviceID,
		DeviceName:   deviceName,
		LastActiveAt: time.Now(),
		CreatedAt:    time.Now(),
	}
	return uc.deviceRepo.Create(ctx, newDevice)
}

func (uc *AuthUseCase) GetUserDevices(ctx context.Context, userID string) ([]domain.Device, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return uc.deviceRepo.List(ctx, uid)
}

func (uc *AuthUseCase) RemoveDevice(ctx context.Context, userID, deviceID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return uc.deviceRepo.Delete(ctx, uid, deviceID)
}
