package grpc_handler

import (
	"auth-service/internal/application"
	authpb "auth-service/pkg/authpb/proto/auth"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	useCase *application.AuthUseCase
}

func NewAuthServer(useCase *application.AuthUseCase) *AuthServer {
	return &AuthServer{
		useCase: useCase,
	}
}

func (s *AuthServer) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}

	userID, err := s.useCase.Register(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &authpb.RegisterResponse{UserId: userID}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password required")
	}

	accessToken, refreshToken, err := s.useCase.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &authpb.LoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *AuthServer) Validate(ctx context.Context, req *authpb.ValidateRequest) (*authpb.ValidateResponse, error) {
	userID, err := s.useCase.ValidateAccess(req.AccessToken)
	if err != nil {
		return &authpb.ValidateResponse{}, status.Error(codes.Unauthenticated, "invalid token")
	}
	return &authpb.ValidateResponse{UserId: userID}, nil
}

func (s *AuthServer) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	access, refresh, err := s.useCase.Refresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "failed to refresh")
	}
	return &authpb.RefreshTokenResponse{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *AuthServer) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	err := s.useCase.Logout(ctx, req.RefreshToken)
	if err != nil {
		return &authpb.LogoutResponse{Success: false}, nil
	}
	return &authpb.LogoutResponse{Success: true}, nil
}
