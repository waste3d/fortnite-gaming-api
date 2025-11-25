package repository

import (
	"context"

	"github.com/waste3d/gameplatform-api/services/course-service/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CourseRepository struct {
	db *gorm.DB
}

func NewCourseRepository(db *gorm.DB) *CourseRepository {
	return &CourseRepository{db: db}
}

func (r *CourseRepository) List(ctx context.Context, search, category string, limit, offset int) ([]domain.Course, int64, error) {
	var courses []domain.Course
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Course{})

	// Фильтр: Поиск по названию (ILIKE = регистронезависимо)
	if search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}
	// Фильтр: Категория
	if category != "" && category != "Все" {
		query = query.Where("category = ?", category)
	}

	// Считаем общее кол-во для пагинации
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Получаем данные
	err := query.Limit(limit).Offset(offset).Order("created_at desc").Find(&courses).Error
	return courses, total, err
}

func (r *CourseRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	var course domain.Course
	err := r.db.WithContext(ctx).First(&course, "id = ?", id).Error
	return &course, err
}

func (r *CourseRepository) Create(ctx context.Context, c *domain.Course) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *CourseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Course{}, "id = ?", id).Error
}

func (r *CourseRepository) CreateLessons(ctx context.Context, lessons []domain.Lesson) error {
	return r.db.WithContext(ctx).Create(&lessons).Error
}

func (r *CourseRepository) GetLessonsByID(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	var course domain.Course
	// Preload("Lessons") заставит GORM сделать второй запрос и заполнить массив
	// Order("lessons.order asc") сортирует уроки
	err := r.db.WithContext(ctx).
		Preload("Lessons", func(db *gorm.DB) *gorm.DB {
			return db.Order("\"order\" asc") // сортировка по полю order
		}).
		First(&course, "id = ?", id).Error
	return &course, err
}
