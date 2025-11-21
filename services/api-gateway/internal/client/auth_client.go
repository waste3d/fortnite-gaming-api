package client

import (
	authpb "api-gateway/pkg/authpb/proto/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	Client authpb.AuthServiceClient
}

func NewAuthClient(url string) (*AuthClient, error) {
	cc, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		return nil, err
	}

	return &AuthClient{
		Client: authpb.NewAuthServiceClient(cc),
	}, nil
}
