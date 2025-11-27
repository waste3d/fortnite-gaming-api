package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/waste3d/gameplatform-api/services/course-service/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type CourseRepository struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewCourseRepository(db *gorm.DB, rdb *redis.Client) *CourseRepository {
	return &CourseRepository{db: db, rdb: rdb}
}

// === КЕШИРУЕМ СПИСОК КУРСОВ ===
func (r *CourseRepository) List(ctx context.Context, search, category string, limit, offset int) ([]domain.Course, int64, error) {
	// Создаем уникальный ключ для кеша на основе фильтров
	key := fmt.Sprintf("courses:list:%s:%s:%d:%d", search, category, limit, offset)

	// 1. Читаем из кеша
	val, err := r.rdb.Get(ctx, key).Result()
	if err == nil {
		var result struct {
			Courses []domain.Course
			Total   int64
		}
		if json.Unmarshal([]byte(val), &result) == nil {
			return result.Courses, result.Total, nil
		}
	}

	// 2. Читаем из БД (если нет в кеше)
	var courses []domain.Course
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Course{})
	if search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}
	if category != "" && category != "Все" {
		query = query.Where("category = ?", category)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err = query.Limit(limit).Offset(offset).Order("created_at desc").Find(&courses).Error
	if err != nil {
		return nil, 0, err
	}

	// 3. Пишем в кеш (на 10 минут, т.к. курсы добавляются не часто)
	cacheData := struct {
		Courses []domain.Course
		Total   int64
	}{courses, total}

	if data, err := json.Marshal(cacheData); err == nil {
		r.rdb.Set(ctx, key, data, 10*time.Minute)
	}

	return courses, total, nil
}

// === КЕШИРУЕМ ОДИН КУРС (С УРОКАМИ) ===
func (r *CourseRepository) GetLessonsByID(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	key := "course:detail:" + id.String()

	// 1. Кеш
	val, err := r.rdb.Get(ctx, key).Result()
	if err == nil {
		var c domain.Course
		if json.Unmarshal([]byte(val), &c) == nil {
			return &c, nil
		}
	}

	// 2. БД
	var course domain.Course
	err = r.db.WithContext(ctx).
		Preload("Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Order("\"order\" asc")
		}).
		First(&course, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	// 3. Сохраняем в кеш на 1 час
	if data, err := json.Marshal(course); err == nil {
		r.rdb.Set(ctx, key, data, 1*time.Hour)
	}

	return &course, err
}

// Инвалидация при изменении курсов (Очищаем все списки, грубо, но надежно)
func (r *CourseRepository) invalidateLists(ctx context.Context) {
	// Удаляем всё, что начинается на courses:list
	// (Redis не умеет удалять по маске просто так, но для простоты в микросервисе можно забить
	// на сложные схемы и просто дать кешу истечь через 10 мин, либо использовать KEYS/SCAN)
	// Для продакшена лучше использовать теги, но здесь TTL 10 минут вполне ок.

	// Удаляем конкретные ключи, если знаем ID, но для списков проще подождать TTL.
}

func (r *CourseRepository) Create(ctx context.Context, c *domain.Course) error {
	// Можно сбросить кеш списков здесь, если критично моментальное появление
	return r.db.WithContext(ctx).Create(c).Error
}

// Остальные методы (Delete, CreateLessons и т.д.) оставляем как есть
func (r *CourseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.rdb.Del(ctx, "course:detail:"+id.String()) // Удаляем кеш детали
	return r.db.WithContext(ctx).Delete(&domain.Course{}, "id = ?", id).Error
}

func (r *CourseRepository) CreateLessons(ctx context.Context, lessons []domain.Lesson) error {
	return r.db.WithContext(ctx).Create(&lessons).Error
}

// GetByID (без уроков, старый метод) можно не кешировать или тоже закешировать
func (r *CourseRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	// ... логика кеша по желанию
	var course domain.Course
	err := r.db.WithContext(ctx).First(&course, "id = ?", id).Error
	return &course, err
}
