package grpctest_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/tomasbasham/grpctest"
	echopb "github.com/tomasbasham/grpctest/testdata/echo/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServer_Connectivity(t *testing.T) {
	t.Parallel()

	s := grpctest.NewServer()
	s.CloseOnCleanup(t)

	echopb.RegisterEchoServiceServer(s, &echoServer{})
	s.Serve()

	conn, err := s.ClientConn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mustEcho(t, conn, "connectivity test")
}

func TestServer_Lifecycle(t *testing.T) {
	t.Parallel()

	t.Run("serve is idempotent", func(t *testing.T) {
		t.Parallel()

		s := grpctest.NewServer()
		s.CloseOnCleanup(t)

		echopb.RegisterEchoServiceServer(s, &echoServer{})

		// First serve.
		s.Serve()

		// Must be safe to call multiple times.
		s.Serve()

		conn, err := s.ClientConn()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mustEcho(t, conn, "serve twice")
	})

	t.Run("close shuts down connections", func(t *testing.T) {
		t.Parallel()

		s := grpctest.NewServer()
		s.CloseOnCleanup(t)

		echopb.RegisterEchoServiceServer(s, &echoServer{})
		s.Serve()

		conn, err := s.ClientConn()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mustEcho(t, conn, "before close")

		s.Close()

		req := &echopb.EchoRequest{
			Message: "after close",
		}

		client := echopb.NewEchoServiceClient(conn)
		_, err = client.Echo(context.Background(), req)
		if err == nil {
			t.Fatal("expected RPC to fail after Close(), but it succeeded")
		}
	})
}

func TestServerWithCustomOptions(t *testing.T) {
	t.Parallel()

	var interceptorCalled bool
	interceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		interceptorCalled = true
		return handler(ctx, req)
	}

	s := grpctest.NewServer(
		grpc.MaxRecvMsgSize(1024*1024*10),
		grpc.UnaryInterceptor(interceptor),
	)
	s.CloseOnCleanup(t)

	echopb.RegisterEchoServiceServer(s, &echoServer{})
	s.Serve()

	conn, err := s.ClientConn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mustEcho(t, conn, "custom options")

	if !interceptorCalled {
		t.Error("interceptor was not called")
	}
}

func TestClientConnOverridesCustomDialer(t *testing.T) {
	t.Parallel()

	s := grpctest.NewServer()
	s.CloseOnCleanup(t)

	echopb.RegisterEchoServiceServer(s, &echoServer{})
	s.Serve()

	badDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return nil, errors.New("this dialer must not be used")
	})

	conn, err := s.ClientConn(
		badDialer,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mustEcho(t, conn, "dialer overridden")
}

func TestClientConnWithCustomInterceptor(t *testing.T) {
	t.Parallel()

	s := grpctest.NewServer()
	s.CloseOnCleanup(t)

	echopb.RegisterEchoServiceServer(s, &echoServer{})
	s.Serve()

	var clientInterceptorCalled bool
	clientInterceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		clientInterceptorCalled = true
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	conn, err := s.ClientConn(grpc.WithUnaryInterceptor(clientInterceptor))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mustEcho(t, conn, "client interceptor")

	if !clientInterceptorCalled {
		t.Error("client interceptor was not called")
	}
}

func TestClientConnContext(t *testing.T) {
	t.Parallel()

	t.Run("succeeds with sufficient timeout", func(t *testing.T) {
		t.Parallel()

		s := grpctest.NewServer()
		s.CloseOnCleanup(t)

		echopb.RegisterEchoServiceServer(s, &echoServer{})
		s.Serve()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		conn, err := s.ClientConnContext(ctx)
		if err != nil {
			t.Fatalf("ClientConnContext failed: %v", err)
		}

		mustEcho(t, conn, "context success")
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()

		s := grpctest.NewServer()
		s.CloseOnCleanup(t)

		echopb.RegisterEchoServiceServer(s, &echoServer{})
		s.Serve()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Ensure context is cancelled.
		time.Sleep(10 * time.Millisecond)

		_, err := s.ClientConnContext(ctx)
		if err == nil {
			t.Fatal("expected error due to cancelled context, got nil")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded, got: %v", err)
		}
	})
}

func TestErr(t *testing.T) {
	t.Parallel()

	t.Run("returns nil on graceful shutdown", func(t *testing.T) {
		t.Parallel()

		s := grpctest.NewServer()
		s.CloseOnCleanup(t)

		echopb.RegisterEchoServiceServer(s, &echoServer{})
		s.Serve()

		errCh := make(chan error, 1)
		go func() {
			errCh <- s.Err()
		}()

		s.Close()

		select {
		case err := <-errCh:
			if err != nil && err.Error() != grpc.ErrServerStopped.Error() {
				t.Errorf("unexpected error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Err() did not return after Close()")
		}
	})

	t.Run("blocks until server stops", func(t *testing.T) {
		t.Parallel()

		s := grpctest.NewServer()
		s.CloseOnCleanup(t)

		echopb.RegisterEchoServiceServer(s, &echoServer{})
		s.Serve()

		errCh := make(chan error, 1)
		returned := make(chan struct{})

		go func() {
			errCh <- s.Err()
			close(returned)
		}()

		// Verify Err() hasn't returned yet
		select {
		case <-returned:
			t.Fatal("Err() returned before Close()")
		case <-time.After(100 * time.Millisecond):
			// Expected behaviour
		}

		s.Close()

		// Now Err() should return
		select {
		case <-returned:
			// Expected
		case <-time.After(5 * time.Second):
			t.Fatal("Err() did not return after Close()")
		}
	})
}

// This test affects global state, so it's isolated and cannot be run in
// parallel.
func TestSetBufferSize(t *testing.T) {
	originalSize := 1 << 20 // 1 MiB default

	defer func() {
		// Reset to original size.
		grpctest.SetBufferSize(originalSize)
	}()

	grpctest.SetBufferSize(2 << 20) // 2 MiB

	s := grpctest.NewServer()
	s.CloseOnCleanup(t)

	echopb.RegisterEchoServiceServer(s, &echoServer{})
	s.Serve()

	conn, err := s.ClientConn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mustEcho(t, conn, "custom buffer size")
}

type echoServer struct {
	echopb.UnimplementedEchoServiceServer
}

func (s *echoServer) Echo(ctx context.Context, in *echopb.EchoRequest) (*echopb.EchoResponse, error) {
	return &echopb.EchoResponse{Message: in.Message}, nil
}

func mustEcho(t *testing.T, conn *grpc.ClientConn, msg string) {
	t.Helper()

	req := &echopb.EchoRequest{
		Message: msg,
	}

	client := echopb.NewEchoServiceClient(conn)
	resp, err := client.Echo(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := resp.Message; got != msg {
		t.Errorf("mismatch:\n  got:  %q\n  want: %q", got, msg)
	}
}
