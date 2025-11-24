package client

import (
	userpb "api-gateway/pkg/userpb/proto/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	Client userpb.UserServiceClient
}

func NewUserClient(url string) (*UserClient, error) {
	cc, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &UserClient{
		Client: userpb.NewUserServiceClient(cc),
	}, nil
}
