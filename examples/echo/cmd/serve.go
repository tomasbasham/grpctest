package cmd

import (
	"net"

	"google.golang.org/grpc"

	"github.com/tomasbasham/grpctest/examples/echo/server"
	echopb "github.com/tomasbasham/grpctest/testdata/echo/v1"
)

func Serve() error {
	l, err := net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}
	defer l.Close()

	s := grpc.NewServer()
	echopb.RegisterEchoServiceServer(s, &server.EchoServer{})

	return s.Serve(l)
}
