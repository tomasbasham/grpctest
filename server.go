package grpctest

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// bufferSize is the default size of the buffered connection.
var bufferSize int32 = 1 << 20 // 1 MiB

// SetBufferSize sets the default buffer size for new servers. Must be called
// before creating any servers. Thread-safe.
func SetBufferSize(size int) {
	if size <= 0 {
		panic("grpctest: buffer size must be positive")
	}
	atomic.StoreInt32(&bufferSize, int32(size))
}

// getBufferSize returns the current buffer size.
func getBufferSize() int {
	return int(atomic.LoadInt32(&bufferSize))
}

// Server is a gRPC server listening on a buffered in-memory connection.
type Server struct {
	*grpc.Server

	listener  *bufconn.Listener
	once      sync.Once
	serveErr  error
	serveDone chan struct{}
}

// NewServer creates a new in-memory test gRPC server. Services must be
// registered before calling [Server.Serve].
func NewServer(opts ...grpc.ServerOption) *Server {
	return &Server{
		Server:    grpc.NewServer(opts...),
		listener:  bufconn.Listen(getBufferSize()),
		serveDone: make(chan struct{}),
	}
}

// Serve begins serving the gRPC server. Safe to call multiple times.
func (s *Server) Serve() {
	s.once.Do(func() {
		go func() {
			s.serveErr = s.Server.Serve(s.listener)
			close(s.serveDone)
		}()
	})
}

// Err blocks until [Server.Serve] completes and returns any error. Useful for
// detecting serve failures in tests.
//
// Example:
//
//	go func() {
//	    if err := s.Err(); err != nil && err != grpc.ErrServerStopped {
//	        t.Errorf("server error: %v", err)
//	    }
//	}()
func (s *Server) Err() error {
	<-s.serveDone
	return s.serveErr
}

// Close shuts down the server and closes the listener.
func (s *Server) Close() {
	s.Stop()
	s.listener.Close()
}

// CloseOnCleanup registers the server to be closed automatically when the test
// ends.
func (s *Server) CloseOnCleanup(t testing.TB) {
	t.Cleanup(s.Close)
}

// ClientConn returns a gRPC client connection to the test server.
//
// The connection is configured to dial the server's in-memory listener.
// Additional [grpc.DialOptions] may be provided but the ContextDialer is fixed
// and cannot be overridden.
func (s *Server) ClientConn(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return s.ClientConnContext(context.Background(), opts...)
}

// ClientConnContext returns a gRPC client connection to the test server.
//
// The connection is configured to dial the server's in-memory listener.
// Additional [grpc.DialOptions] may be provided but the ContextDialer is fixed
// and cannot be overridden.
func (s *Server) ClientConnContext(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, opts...)

	// Use a custom dialer that dials the bufconn listener.
	opts = append(opts, grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return s.listener.Dial()
	}))

	conn, err := grpc.NewClient("passthrough:///test", opts...)
	if err != nil {
		return nil, err
	}

	// Drive the connection out of idle and wait until ready.
	conn.Connect()

	connCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return conn, nil
		}
		if !conn.WaitForStateChange(connCtx, state) {
			return nil, connCtx.Err()
		}
	}
}
