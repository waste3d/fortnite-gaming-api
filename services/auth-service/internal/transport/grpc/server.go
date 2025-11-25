package grpc_server

import (
	"context"

	"auth-service/internal/infrastructure/application/usecase"

	authpb "auth-service/pkg/authpb/proto/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	useCase *usecase.AuthUseCase
}

func NewAuthServer(uc *usecase.AuthUseCase) *AuthServer {
	return &AuthServer{useCase: uc}
}

func (s *AuthServer) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	userID, err := s.useCase.Register(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &authpb.RegisterResponse{UserId: userID}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	access, refresh, err := s.useCase.Login(ctx, req.Email, req.Password, req.DeviceId, req.DeviceName)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &authpb.LoginResponse{AccessToken: access, RefreshToken: refresh}, nil
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
	_ = s.useCase.Logout(ctx, req.RefreshToken, req.DeviceId)
	return &authpb.LogoutResponse{Success: true}, nil
}

func (s *AuthServer) ForgotPassword(ctx context.Context, req *authpb.ForgotPasswordRequest) (*authpb.ForgotPasswordResponse, error) {
	err := s.useCase.ForgotPassword(ctx, req.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to process request")
	}
	return &authpb.ForgotPasswordResponse{Success: true}, nil
}

func (s *AuthServer) ResetPassword(ctx context.Context, req *authpb.ResetPasswordRequest) (*authpb.ResetPasswordResponse, error) {
	err := s.useCase.ResetPassword(ctx, req.Token, req.NewPassword)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &authpb.ResetPasswordResponse{Success: true}, nil
}

func (s *AuthServer) RequestEmailChange(ctx context.Context, req *authpb.RequestEmailChangeRequest) (*authpb.RequestEmailChangeResponse, error) {
	err := s.useCase.RequestEmailChange(ctx, req.UserId, req.NewEmail)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &authpb.RequestEmailChangeResponse{Success: true, Message: "Confirmation email sent"}, nil
}

func (s *AuthServer) ConfirmEmailChange(ctx context.Context, req *authpb.ConfirmEmailChangeRequest) (*authpb.ConfirmEmailChangeResponse, error) {
	err := s.useCase.ConfirmEmailChange(ctx, req.Token)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &authpb.ConfirmEmailChangeResponse{Success: true, Message: "Email updated"}, nil
}

func (s *AuthServer) GetDevices(ctx context.Context, req *authpb.GetDevicesRequest) (*authpb.GetDevicesResponse, error) {
	devices, err := s.useCase.GetUserDevices(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var pbDevices []*authpb.DeviceInfo
	for _, d := range devices {
		pbDevices = append(pbDevices, &authpb.DeviceInfo{
			Id:         d.DeviceID,
			DeviceName: d.DeviceName,
			LastActive: d.LastActiveAt.String(),
		})
	}

	return &authpb.GetDevicesResponse{Devices: pbDevices}, nil
}

func (s *AuthServer) RemoveDevice(ctx context.Context, req *authpb.RemoveDeviceRequest) (*authpb.RemoveDeviceResponse, error) {
	err := s.useCase.RemoveDevice(ctx, req.UserId, req.DeviceId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &authpb.RemoveDeviceResponse{Success: true}, nil
}
