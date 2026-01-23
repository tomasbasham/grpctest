package grpctest_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/tomasbasham/grpctest"
	echopb "github.com/tomasbasham/grpctest/testdata/echo/v1"
)

func Example_integration() {
	s := grpctest.NewServer()
	defer s.Stop()

	echopb.RegisterEchoServiceServer(s, &echoServer{})
	s.Serve()

	conn, err := s.ClientConn()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create connection: %v", err)
		return
	}

	client := echopb.NewEchoServiceClient(conn)
	resp, err := client.Echo(context.Background(), &echopb.EchoRequest{Message: "Hello, world"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v", err)
		return
	}

	fmt.Println(resp.GetMessage())
	// Output:
	// Hello, world
}

func ExampleServer_Err() {
	s := grpctest.NewServer()
	s.Serve()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	fmt.Println("waiting for server to stop...")
	if err := s.Err(); err != nil && err != grpc.ErrServerStopped {
		// Handle server error.
	}
	fmt.Println("server stopped")
	// Output:
	// waiting for server to stop...
	// server stopped
}
