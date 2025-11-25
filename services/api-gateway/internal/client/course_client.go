package client

import (
	coursepb "api-gateway/pkg/coursepb/proto/course"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type CourseClient struct {
	Client coursepb.CourseServiceClient
}

func NewCourseClient(url string) (*CourseClient, error) {
	cc, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &CourseClient{
		Client: coursepb.NewCourseServiceClient(cc),
	}, nil
}
