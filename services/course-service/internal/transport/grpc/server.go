package grpc_server

import (
	"context"

	"github.com/waste3d/gameplatform-api/services/course-service/internal/domain"
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

// GetCourse: возвращает детали и ссылку, ЕСЛИ есть доступ
func (s *CourseServer) GetCourse(ctx context.Context, req *coursepb.GetCourseRequest) (*coursepb.GetCourseResponse, error) {
	uid, err := uuid.Parse(req.CourseId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid course id")
	}

	course, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.NotFound, "course not found")
	}

	// === ЛОГИКА ПРОВЕРКИ ДОСТУПА ===
	hasAccess := false

	// Если ID пользователя пришел (он залогинен)
	if req.UserId != "" {
		// Спрашиваем у User Service статус подписки
		userRes, err := s.userClient.GetProfile(ctx, &userpb.GetProfileRequest{UserId: req.UserId})

		if err == nil {
			status := userRes.SubscriptionStatus
			// Условие доступа: admin или Персональный
			if status == "admin" || status == "Персональный" {
				hasAccess = true
			}
		}
	}

	cloudLink := ""
	if hasAccess {
		cloudLink = course.CloudLink
	}

	return &coursepb.GetCourseResponse{
		Course: &coursepb.Course{
			Id:          course.ID.String(),
			Title:       course.Title,
			Description: course.Description,
			Category:    course.Category,
			Duration:    course.Duration,
			CoverUrl:    course.CoverURL,
			CloudLink:   cloudLink, // Заполнено только если hasAccess=true
		},
		HasAccess: hasAccess,
	}, nil
}

// CreateCourse: создание (без проверок прав, предполагается вызов админом)
func (s *CourseServer) CreateCourse(ctx context.Context, req *coursepb.CreateCourseRequest) (*coursepb.CreateCourseResponse, error) {
	course := &domain.Course{
		ID:          uuid.New(),
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Duration:    req.Duration,
		CoverURL:    req.CoverUrl,
		CloudLink:   req.CloudLink,
	}

	if err := s.repo.Create(ctx, course); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &coursepb.CreateCourseResponse{Id: course.ID.String()}, nil
}

func (s *CourseServer) DeleteCourse(ctx context.Context, req *coursepb.DeleteCourseRequest) (*coursepb.DeleteCourseResponse, error) {
	uid, _ := uuid.Parse(req.Id)
	_ = s.repo.Delete(ctx, uid)
	return &coursepb.DeleteCourseResponse{Success: true}, nil
}
