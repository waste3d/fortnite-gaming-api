package grpc_server

import (
	"context"

	"github.com/waste3d/gameplatform-api/services/user-service/internal/domain"
	"github.com/waste3d/gameplatform-api/services/user-service/internal/infrastructure/repository"

	userpb "github.com/waste3d/gameplatform-api/services/user-service/pkg/userpb/proto/user"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServer struct {
	userpb.UnimplementedUserServiceServer
	repo *repository.ProfileRepository
}

func NewUserServer(repo *repository.ProfileRepository) *UserServer {
	return &UserServer{repo: repo}
}

func (s *UserServer) CreateProfile(ctx context.Context, req *userpb.CreateProfileRequest) (*userpb.CreateProfileResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user ID: %v", err)
	}

	profile := &domain.Profile{
		ID:       uid,
		Email:    req.Email,
		Username: req.Username,
	}

	if err := s.repo.Create(ctx, profile); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create profile: %v", err)
	}

	return &userpb.CreateProfileResponse{ProfileId: profile.ID.String()}, nil
}

func (s *UserServer) SyncEmail(ctx context.Context, req *userpb.SyncEmailRequest) (*userpb.SyncEmailResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	err := s.repo.UpdateEmail(ctx, uid, req.NewEmail)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to sync email")
	}
	return &userpb.SyncEmailResponse{Success: true}, nil
}

func (s *UserServer) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.UpdateProfileResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	// Валидация
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username cannot be empty")
	}

	// Вызываем репозиторий
	err = s.repo.UpdateUsername(ctx, uid, req.Username)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update profile")
	}

	return &userpb.UpdateProfileResponse{Success: true}, nil
}

func (s *UserServer) SetAvatar(ctx context.Context, req *userpb.SetAvatarRequest) (*userpb.SetAvatarResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	// Проверяем диапазон (у нас 10 пресетов)
	if req.AvatarId < 1 || req.AvatarId > 10 {
		return nil, status.Error(codes.InvalidArgument, "avatar_id must be between 1 and 10")
	}

	// Вызываем репозиторий
	err = s.repo.UpdateAvatar(ctx, uid, int(req.AvatarId))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update avatar")
	}

	return &userpb.SetAvatarResponse{
		Success:  true,
		AvatarId: req.AvatarId,
	}, nil
}

func (s *UserServer) StartCourse(ctx context.Context, req *userpb.StartCourseRequest) (*userpb.StartCourseResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	uc := &domain.UserCourse{
		UserID:   uid,
		CourseID: req.CourseId,
		Title:    req.Title,
		CoverURL: req.CoverUrl,
	}

	if err := s.repo.StartCourse(ctx, uc); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to start course: %v", err)
	}

	return &userpb.StartCourseResponse{Success: true}, nil
}

func (s *UserServer) UpdateProgress(ctx context.Context, req *userpb.UpdateProgressRequest) (*userpb.UpdateProgressResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	newStatus, err := s.repo.UpdateProgress(ctx, uid, req.CourseId, req.ProgressPercent)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update progress: %v", err)
	}

	return &userpb.UpdateProgressResponse{Success: true, Status: newStatus}, nil
}

func (s *UserServer) GetProfile(ctx context.Context, req *userpb.GetProfileRequest) (*userpb.GetProfileResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	profile, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}

	// 1. Получаем курсы из репозитория
	userCourses, err := s.repo.GetUserCourses(ctx, uid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user courses: %v", err)
	}

	// Ошибку можно залогировать, но не прерывать запрос профиля

	active := []*userpb.CoursePreview{}
	completed := []*userpb.CoursePreview{}

	// 2. Раскладываем по спискам
	for _, c := range userCourses {
		pb := &userpb.CoursePreview{
			Id:              c.CourseID,
			Title:           c.Title,
			ProgressPercent: c.ProgressPercent,
			CoverUrl:        c.CoverURL,
		}
		if c.Status == "completed" {
			completed = append(completed, pb)
		} else {
			active = append(active, pb)
		}
	}

	return &userpb.GetProfileResponse{
		Id:                 profile.ID.String(),
		Email:              profile.Email,
		Username:           profile.Username,
		AvatarId:           int32(profile.AvatarID),
		SubscriptionStatus: profile.SubscriptionStatus,
		ActiveCourses:      active,    // <-- Теперь тут данные
		CompletedCourses:   completed, // <-- И тут
	}, nil
}

func (s *UserServer) CompleteLesson(ctx context.Context, req *userpb.CompleteLessonRequest) (*userpb.CompleteLessonResponse, error) {
	uid, _ := uuid.Parse(req.UserId)

	// 1. Сохраняем урок как пройденный
	err := s.repo.AddCompletedLesson(ctx, &domain.CompletedLesson{
		UserID:   uid,
		CourseID: req.CourseId,
		LessonID: req.LessonId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to save lesson")
	}

	// 2. Считаем, сколько всего пройдено
	completedCount, _ := s.repo.CountCompletedLessons(ctx, uid, req.CourseId)

	// 3. Считаем процент (Backend logic!)
	var percent int32
	if req.TotalLessons > 0 {
		percent = int32((float64(completedCount) / float64(req.TotalLessons)) * 100)
	}
	if percent > 100 {
		percent = 100
	}

	// 4. Обновляем UserCourse (процент и статус)
	// Используем ту логику с защитой, которую мы писали в прошлом ответе
	newStatus, err := s.repo.UpdateProgress(ctx, uid, req.CourseId, percent)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update progress")
	}

	return &userpb.CompleteLessonResponse{
		Success:    true,
		NewPercent: percent,
		Status:     newStatus,
	}, nil
}

func (s *UserServer) GetCompletedLessons(ctx context.Context, req *userpb.GetCompletedLessonsRequest) (*userpb.GetCompletedLessonsResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	ids, err := s.repo.GetCompletedLessonIDs(ctx, uid, req.CourseId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get lessons")
	}
	return &userpb.GetCompletedLessonsResponse{LessonIds: ids}, nil
}
