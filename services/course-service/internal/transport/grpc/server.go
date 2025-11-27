package grpc_server

import (
	"context"
	"log"

	"github.com/waste3d/gameplatform-api/services/course-service/internal/domain"
	"github.com/waste3d/gameplatform-api/services/course-service/internal/infrastructure/parser"
	"github.com/waste3d/gameplatform-api/services/course-service/internal/infrastructure/repository"
	coursepb "github.com/waste3d/gameplatform-api/services/course-service/pkg/coursepb/proto/course"

	userpb "github.com/waste3d/gameplatform-api/services/user-service/pkg/userpb/proto/user"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CourseServer struct {
	coursepb.UnimplementedCourseServiceServer
	repo       *repository.CourseRepository
	userClient userpb.UserServiceClient
}

func NewCourseServer(repo *repository.CourseRepository, uc userpb.UserServiceClient) *CourseServer {
	return &CourseServer{repo: repo, userClient: uc}
}

// GetCourses: возвращает список БЕЗ секретных ссылок
func (s *CourseServer) GetCourses(ctx context.Context, req *coursepb.GetCoursesRequest) (*coursepb.GetCoursesResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}
	offset := int(req.Offset)

	courses, total, err := s.repo.List(ctx, req.Search, req.Category, limit, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}

	var pbCourses []*coursepb.Course
	for _, c := range courses {
		pbCourses = append(pbCourses, &coursepb.Course{
			Id:          c.ID.String(),
			Title:       c.Title,
			Description: c.Description,
			Category:    c.Category,
			Duration:    c.Duration,
			CoverUrl:    c.CoverURL,
			CloudLink:   "", // В списке ссылку скрываем всегда
		})
	}

	return &coursepb.GetCoursesResponse{Courses: pbCourses, TotalCount: total}, nil
}

func (s *CourseServer) GetCourse(ctx context.Context, req *coursepb.GetCourseRequest) (*coursepb.GetCourseResponse, error) {
	uid, err := uuid.Parse(req.CourseId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid course id")
	}

	// Теперь этот метод вернет курс ВМЕСТЕ с уроками благодаря Preload
	course, err := s.repo.GetLessonsByID(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.NotFound, "course not found")
	}

	// === ПРОВЕРКА ДОСТУПА ===
	hasAccess := false
	if req.UserId != "" {
		userRes, err := s.userClient.GetProfile(ctx, &userpb.GetProfileRequest{UserId: req.UserId})
		if err == nil {
			// 1. Глобальный доступ (Админ или Персональный/Безлимит)
			// Если CourseLimit == -1, считаем это полным доступом
			if userRes.SubscriptionStatus == "admin" {
				hasAccess = true
			} else {
				// 2. Проверяем, есть ли этот курс в списке ActiveCourses у пользователя
				// (Это значит, что он потратил на него слот)
				for _, activeCourse := range userRes.ActiveCourses {
					if activeCourse.Id == req.CourseId {
						hasAccess = true
						break
					}
				}
				// Также проверяем завершенные
				if !hasAccess {
					for _, completedCourse := range userRes.CompletedCourses {
						if completedCourse.Id == req.CourseId {
							hasAccess = true
							break
						}
					}
				}
			}
		}
	}

	// Формируем ответ
	resp := &coursepb.GetCourseResponse{
		Course: &coursepb.Course{
			Id:          course.ID.String(),
			Title:       course.Title,
			Description: course.Description,
			Category:    course.Category,
			Duration:    course.Duration,
			CoverUrl:    course.CoverURL,
			CloudLink:   "", // Прячем общую ссылку, если хотим заставить кликать по урокам
		},
		HasAccess: hasAccess,
		Lessons:   []*coursepb.Lesson{}, // Инициализируем пустым
	}

	// Если есть доступ — отдаем ссылку и уроки
	if hasAccess {
		resp.Course.CloudLink = course.CloudLink

		completedIDs := make(map[string]bool)

		if req.UserId != "" {
			// Вызов нового метода GetCompletedLessons (Исправлена опечатка и тип запроса)
			res, err := s.userClient.GetCompletedLessons(ctx, &userpb.GetCompletedLessonsRequest{
				UserId:   req.UserId,
				CourseId: course.ID.String(),
			})
			if err == nil {
				for _, id := range res.LessonIds {
					completedIDs[id] = true
				}
			}
		}

		// Конвертируем доменные уроки в protobuf
		for _, l := range course.Lessons {
			resp.Lessons = append(resp.Lessons, &coursepb.Lesson{
				Id:        l.ID.String(),
				Title:     l.Title,
				FileLink:  l.FileLink,
				Order:     int32(l.Order),
				Completed: completedIDs[l.ID.String()],
			})
		}
	}

	return resp, nil
}

func (s *CourseServer) CreateCourse(ctx context.Context, req *coursepb.CreateCourseRequest) (*coursepb.CreateCourseResponse, error) {
	courseID := uuid.New()

	course := &domain.Course{
		ID:          courseID,
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Duration:    req.Duration,
		CoverURL:    req.CoverUrl,
		CloudLink:   req.CloudLink,
	}

	// 1. Сохраняем курс
	if err := s.repo.Create(ctx, course); err != nil {
		return nil, status.Error(codes.Internal, "failed to create course")
	}

	// 2. Если есть ссылка на облако — ПАРСИМ
	if req.CloudLink != "" {
		// Запускаем в горутине, чтобы не тормозить ответ админке?
		// Лучше синхронно, чтобы админ сразу понял, если ссылка битая.
		lessonsDTO, err := parser.ParseFolder(req.CloudLink)

		if err != nil {
			log.Printf("Warning: failed to parse cloud link: %v", err)
			// Не возвращаем ошибку клиенту, курс-то создан. Просто лог.
		} else {
			var lessons []domain.Lesson
			for i, dto := range lessonsDTO {
				lessons = append(lessons, domain.Lesson{
					ID:       uuid.New(),
					CourseID: courseID,
					Title:    dto.Title,
					FileLink: dto.FileLink,
					Order:    i + 1,
				})
			}
			// Сохраняем пачкой
			if len(lessons) > 0 {
				_ = s.repo.CreateLessons(ctx, lessons)
			}
		}
	}

	return &coursepb.CreateCourseResponse{Id: course.ID.String()}, nil
}

func (s *CourseServer) DeleteCourse(ctx context.Context, req *coursepb.DeleteCourseRequest) (*coursepb.DeleteCourseResponse, error) {
	uid, _ := uuid.Parse(req.Id)
	_ = s.repo.Delete(ctx, uid)
	return &coursepb.DeleteCourseResponse{Success: true}, nil
}
