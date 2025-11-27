package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/waste3d/gameplatform-api/services/user-service/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ProfileRepository struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewProfileRepository(db *gorm.DB, rdb *redis.Client) *ProfileRepository {
	return &ProfileRepository{db: db, rdb: rdb}
}

// Вспомогательная функция для очистки кеша
func (r *ProfileRepository) invalidateCache(ctx context.Context, userID string) {
	r.rdb.Del(ctx, "profile:"+userID)
}

// === GET BY ID С КЕШЕМ ===
func (r *ProfileRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	key := "profile:" + id.String()

	// 1. Пытаемся взять из Redis
	val, err := r.rdb.Get(ctx, key).Result()
	if err == nil {
		var profile domain.Profile
		if json.Unmarshal([]byte(val), &profile) == nil {
			return &profile, nil
		}
	}

	// 2. Если нет в кеше - берем из БД
	var profile domain.Profile
	err = r.db.WithContext(ctx).Where("id = ?", id).First(&profile).Error
	if err != nil {
		return nil, err
	}

	// 3. Сохраняем в Redis на 1 час
	if data, err := json.Marshal(profile); err == nil {
		r.rdb.Set(ctx, key, data, 1*time.Hour)
	}

	return &profile, nil
}

// === МЕТОДЫ С ИНВАЛИДАЦИЕЙ (Сбросом) КЕША ===

func (r *ProfileRepository) Create(ctx context.Context, profile *domain.Profile) error {
	// Кеш сбрасывать не надо, профиля еще нет, но на всякий случай
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *ProfileRepository) Update(ctx context.Context, profile *domain.Profile) error {
	err := r.db.WithContext(ctx).Save(profile).Error
	if err == nil {
		r.invalidateCache(ctx, profile.ID.String())
	}
	return err
}

func (r *ProfileRepository) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
	err := r.db.WithContext(ctx).Model(&domain.Profile{}).Where("id = ?", id).Update("email", email).Error
	if err == nil {
		r.invalidateCache(ctx, id.String())
	}
	return err
}

func (r *ProfileRepository) UpdateUsername(ctx context.Context, id uuid.UUID, username string) error {
	err := r.db.WithContext(ctx).Model(&domain.Profile{}).Where("id = ?", id).Update("username", username).Error
	if err == nil {
		r.invalidateCache(ctx, id.String())
	}
	return err
}

func (r *ProfileRepository) UpdateAvatar(ctx context.Context, id uuid.UUID, avatarID int) error {
	err := r.db.WithContext(ctx).Model(&domain.Profile{}).Where("id = ?", id).Update("avatar_id", avatarID).Error
	if err == nil {
		r.invalidateCache(ctx, id.String())
	}
	return err
}

func (r *ProfileRepository) UpdateSubscription(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error {
	err := r.db.WithContext(ctx).Model(&domain.Profile{}).Where("id = ?", userID).Updates(updates).Error
	if err == nil {
		r.invalidateCache(ctx, userID.String())
	}
	return err
}

// Остальные методы (StartCourse, UpdateProgress, GetUserCourses и т.д.) можно пока не трогать,
// так как они часто меняются и их кеширование сложнее. Но профиль — это 80% успеха.

func (r *ProfileRepository) StartCourse(ctx context.Context, uc *domain.UserCourse) error {
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
	var existing domain.UserCourse
	err := r.db.WithContext(ctx).Where("user_id = ? AND course_id = ?", userID, courseID).First(&existing).Error
	if err != nil {
		return "", err
	}

	if existing.Status == "completed" {
		r.db.WithContext(ctx).Model(&existing).Update("last_accessed_at", time.Now())
		return "completed", nil
	}
	if percent <= existing.ProgressPercent {
		r.db.WithContext(ctx).Model(&existing).Update("last_accessed_at", time.Now())
		return existing.Status, nil
	}

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
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("last_accessed_at desc").Find(&courses).Error
	return courses, err
}

func (r *ProfileRepository) AddCompletedLesson(ctx context.Context, item *domain.CompletedLesson) error {
	return r.db.WithContext(ctx).FirstOrCreate(item).Error
}

func (r *ProfileRepository) CountCompletedLessons(ctx context.Context, userID uuid.UUID, courseID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.CompletedLesson{}).Where("user_id = ? AND course_id = ?", userID, courseID).Count(&count).Error
	return count, err
}

func (r *ProfileRepository) GetCompletedLessonIDs(ctx context.Context, userID uuid.UUID, courseID string) ([]string, error) {
	var lessons []domain.CompletedLesson
	err := r.db.WithContext(ctx).Where("user_id = ? AND course_id = ?", userID, courseID).Find(&lessons).Error
	var ids []string
	for _, l := range lessons {
		ids = append(ids, l.LessonID)
	}
	return ids, err
}

func (r *ProfileRepository) CountUserCourses(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.UserCourse{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *ProfileRepository) UserHasCourse(ctx context.Context, userID uuid.UUID, courseID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.UserCourse{}).Where("user_id = ? AND course_id = ?", userID, courseID).Count(&count).Error
	return count > 0, err
}
