package repository

import (
	"context"
	"encoding/json"
	"fmt"
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
	err = r.db.WithContext(ctx).Preload("UnlockedAvatars").Where("id = ?", id).First(&profile).Error
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

// Возвращаем bool (created) и error
func (r *ProfileRepository) AddCompletedLesson(ctx context.Context, item *domain.CompletedLesson) (bool, error) {
	result := r.db.WithContext(ctx).FirstOrCreate(item)
	if result.Error != nil {
		return false, result.Error
	}
	// RowsAffected == 1, если запись была создана. 0, если уже существовала.
	return result.RowsAffected > 0, nil
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

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func (r *ProfileRepository) CheckAndIncrementStreak(ctx context.Context, userID uuid.UUID) (bool, error) {
	var profile domain.Profile
	if err := r.db.WithContext(ctx).Where("id = ?", userID).First(&profile).Error; err != nil {
		return false, err
	}

	now := time.Now().UTC()
	today := truncateToDay(now)
	lastActivity := truncateToDay(profile.LastStreakAt.UTC())

	daysDiff := int(today.Sub(lastActivity).Hours() / 24)

	updates := make(map[string]interface{})

	if daysDiff == 0 {
		// Активность сегодня уже была, стрик не обновляем.
		return false, nil
	} else if daysDiff == 1 {
		// Продолжаем стрик
		updates["streak"] = profile.Streak + 1
		updates["last_streak_at"] = now
	} else {
		// Сбрасываем стрик до 1
		updates["streak"] = 1
		updates["last_streak_at"] = now
	}

	if err := r.db.WithContext(ctx).Model(&profile).Updates(updates).Error; err != nil {
		return false, err
	}

	r.invalidateCache(ctx, userID.String())
	// Возвращаем true, потому что стрик был обновлен
	return true, nil
}

func (r *ProfileRepository) AddCourseSlots(ctx context.Context, userID uuid.UUID, count int) error {
	err := r.db.WithContext(ctx).Model(&domain.Profile{}).
		Where("id = ?", userID).
		Update("course_limit", gorm.Expr("course_limit + ?", count)).Error
	if err == nil {
		r.invalidateCache(ctx, userID.String())
	}
	return err
}

func (r *ProfileRepository) ChangeBalance(ctx context.Context, userID uuid.UUID, amount int) (int, error) {
	var newBalance int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p domain.Profile
		if err := tx.Select("balance").Where("id = ?", userID).First(&p).Error; err != nil {
			return err
		}

		if p.Balance+amount < 0 {
			return fmt.Errorf("insufficient funds")
		}

		p.Balance += amount
		newBalance = p.Balance

		return tx.Model(&domain.Profile{}).Where("id = ?", userID).Update("balance", p.Balance).Error
	})

	if err == nil {
		r.invalidateCache(ctx, userID.String())
	}
	return newBalance, err
}

func (r *ProfileRepository) AddUnlockedAvatar(ctx context.Context, userID uuid.UUID, avatarID int) (bool, error) {
	// Проверяем, есть ли уже
	var count int64
	r.db.WithContext(ctx).Model(&domain.UnlockedAvatar{}).
		Where("user_id = ? AND avatar_id = ?", userID, avatarID).
		Count(&count)

	if count > 0 {
		return true, nil // Уже есть
	}

	err := r.db.WithContext(ctx).Create(&domain.UnlockedAvatar{
		UserID:   userID,
		AvatarID: avatarID,
	}).Error

	if err == nil {
		r.invalidateCache(ctx, userID.String())
	}
	return false, err
}

func (r *ProfileRepository) GetLeaderboard(ctx context.Context, limit int) ([]domain.Profile, error) {
	var users []domain.Profile
	// Сортируем по стрику и кол-ву пройденных курсов
	err := r.db.WithContext(ctx).
		Order("streak desc, completed_count desc").
		Limit(limit).
		Find(&users).Error
	return users, err
}

func (r *ProfileRepository) IncrementCompletedCount(ctx context.Context, userID uuid.UUID) {
	r.db.WithContext(ctx).Model(&domain.Profile{}).
		Where("id = ?", userID).
		Update("completed_count", gorm.Expr("completed_count + 1"))
}

// GetUserCourseStatus возвращает текущий статус курса ("active" или "completed")
func (r *ProfileRepository) GetUserCourseStatus(ctx context.Context, userID uuid.UUID, courseID string) (string, error) {
	var uc domain.UserCourse
	// Выбираем только поле status для оптимизации
	err := r.db.WithContext(ctx).
		Select("status").
		Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&uc).Error

	if err != nil {
		return "", err
	}
	return uc.Status, nil
}

func (r *ProfileRepository) GetUserRank(ctx context.Context, userID uuid.UUID) (int64, error) {
	var rank int64

	subQuery := r.db.WithContext(ctx).Model(&domain.Profile{}).Select("id, ROW_NUMBER() over (order by streak desc, completed_count desc, balance desc) as rank")

	err := r.db.WithContext(ctx).
		Table("(?) as ranked_users", subQuery).
		Select("rank").
		Where("id = ?", userID).
		Row().Scan(&rank)

	if err != nil {
		return 0, err
	}

	return rank, nil
}
