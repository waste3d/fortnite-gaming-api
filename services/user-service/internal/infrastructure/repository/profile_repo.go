package repository

import (
	"context"
	"time"

	"github.com/waste3d/gameplatform-api/services/user-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProfileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) Create(ctx context.Context, profile *domain.Profile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *ProfileRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	var profile domain.Profile
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&profile).Error
	return &profile, err
}

func (r *ProfileRepository) Update(ctx context.Context, profile *domain.Profile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

func (r *ProfileRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
	return r.db.WithContext(ctx).Model(&domain.Profile{}).
		Where("id = ?", id).
		Update("email", email).Error
}

func (r *ProfileRepository) UpdateUsername(ctx context.Context, id uuid.UUID, username string) error {
	result := r.db.WithContext(ctx).Model(&domain.Profile{}).
		Where("id = ?", id).
		Update("username", username)

	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Обновляем только AvatarID
func (r *ProfileRepository) UpdateAvatar(ctx context.Context, id uuid.UUID, avatarID int) error {
	return r.db.WithContext(ctx).Model(&domain.Profile{}).
		Where("id = ?", id).
		Update("avatar_id", avatarID).Error
}

func (r *ProfileRepository) StartCourse(ctx context.Context, uc *domain.UserCourse) error {
	// FirstOrCreate чтобы не дублировать, если юзер нажал кнопку дважды
	return r.db.WithContext(ctx).
		Where(domain.UserCourse{UserID: uc.UserID, CourseID: uc.CourseID}).
		Attrs(domain.UserCourse{
			Title:          uc.Title,
			CoverURL:       uc.CoverURL,
			Status:         "active",
			LastAccessedAt: time.Now(),
			CreatedAt:      time.Now(),
		}).
		FirstOrCreate(uc).Error
}

func (r *ProfileRepository) UpdateProgress(ctx context.Context, userID uuid.UUID, courseID string, percent int32) (string, error) {
	// 1. Сначала получаем текущее состояние записи в БД
	var existing domain.UserCourse
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&existing).Error

	if err != nil {
		return "", err
	}

	// 2. ЗАЩИТА: Если курс уже завершен, мы НЕ даем сбросить статус обратно в active
	if existing.Status == "completed" {
		// Просто обновляем время последнего доступа, чтобы курс поднялся в списке
		r.db.WithContext(ctx).Model(&existing).Update("last_accessed_at", time.Now())
		return "completed", nil
	}

	// 3. ЗАЩИТА: Если новый процент МЕНЬШЕ уже сохраненного (например, зашли с нового устройства),
	// мы не обновляем прогресс вниз.
	if percent <= existing.ProgressPercent {
		r.db.WithContext(ctx).Model(&existing).Update("last_accessed_at", time.Now())
		return existing.Status, nil
	}

	// 4. Если всё ок (прогресс растет), обновляем
	status := "active"
	if percent >= 100 {
		status = "completed"
		percent = 100
	}

	err = r.db.WithContext(ctx).Model(&domain.UserCourse{}).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		Updates(map[string]interface{}{
			"progress_percent": percent,
			"status":           status,
			"last_accessed_at": time.Now(),
		}).Error

	return status, err
}

func (r *ProfileRepository) GetUserCourses(ctx context.Context, userID uuid.UUID) ([]domain.UserCourse, error) {
	var courses []domain.UserCourse
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_accessed_at desc"). // Сначала последние открытые
		Find(&courses).Error
	return courses, err
}
