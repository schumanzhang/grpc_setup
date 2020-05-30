package protoservices

import (
	"context"

	"grpc_setup/proto"

	"github.com/golang/protobuf/ptypes/empty"
)

type healthService struct{}

func (service *healthService) HealthCheck(context.Context, *empty.Empty) (*proto.HealthStatus, error) {
	return &proto.HealthStatus{Status: "OK"}, nil
}

// NewHealthService - Returns new implementation of health service
func NewHealthService() proto.HealthServer {
	return new(healthService)
}
