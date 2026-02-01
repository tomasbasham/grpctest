package server

import (
	"context"

	echopb "github.com/tomasbasham/grpctest/testdata/echo/v1"
)

type EchoServer struct {
	echopb.UnimplementedEchoServiceServer
}

func (s *EchoServer) Echo(ctx context.Context, in *echopb.EchoRequest) (*echopb.EchoResponse, error) {
	return &echopb.EchoResponse{Message: in.Message}, nil
}
