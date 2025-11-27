package client

import (
	paymentpb "api-gateway/pkg/paymentpb/proto/payment"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PaymentClient struct {
	Client paymentpb.PaymentServiceClient
}

func NewPaymentClient(url string) (*PaymentClient, error) {
	cc, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &PaymentClient{
		Client: paymentpb.NewPaymentServiceClient(cc),
	}, nil
}
