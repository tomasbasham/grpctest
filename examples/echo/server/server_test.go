package server_test

import (
	"context"
	"testing"

	"github.com/tomasbasham/grpctest"
	echopb "github.com/tomasbasham/grpctest/testdata/echo/v1"

	"github.com/tomasbasham/grpctest/examples/echo/server"
)

func TestEcho(t *testing.T) {
	s := grpctest.NewServer()
	s.CloseOnCleanup(t)

	echopb.RegisterEchoServiceServer(s, &server.EchoServer{})
	s.Serve()

	conn, err := s.ClientConn()
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	// The message to echo.
	want := "Hello, world"

	client := echopb.NewEchoServiceClient(conn)
	resp, err := client.Echo(context.Background(), &echopb.EchoRequest{Message: want})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := resp.Message; got != want {
		t.Errorf("mismatch:\n  got:  %q\n  want: %q", got, want)
	}
}
