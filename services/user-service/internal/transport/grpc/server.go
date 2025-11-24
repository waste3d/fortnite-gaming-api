package grpc_server

import (
	"context"
	"user-service/internal/domain"
	"user-service/internal/infrastructure/repository"

	userpb "user-service/pkg/userpb/proto/user"

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

func (s *UserServer) GetProfile(ctx context.Context, req *userpb.GetProfileRequest) (*userpb.GetProfileResponse, error) {
	uid, _ := uuid.Parse(req.UserId)
	profile, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.NotFound, "profile not found")
	}

	// TODO: Get active and completed courses from course service
	active := []*userpb.CoursePreview{}
	completed := []*userpb.CoursePreview{}

	return &userpb.GetProfileResponse{
		Id:                 profile.ID.String(),
		Email:              profile.Email,
		Username:           profile.Username,
		AvatarId:           int32(profile.AvatarID),
		SubscriptionStatus: profile.SubscriptionStatus,
		ActiveCourses:      active,
		CompletedCourses:   completed,
	}, nil
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
