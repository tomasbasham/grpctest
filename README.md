# grpctest ![test](https://github.com/tomasbasham/grpctest/workflows/test/badge.svg)

A Go module for testing gRPC servers through in-process integration tests. It
provides a test server that communicates over an in-memory buffer rather than
network sockets, enabling fast, isolated tests that exercise the complete gRPC
stack without requiring port allocation or network configuration.

## Motivation

Testing gRPC services typically requires managing network listeners, port
allocation, and connection lifecycles. This boilerplate distracts from test
logic and introduces flakiness from network-level issues. `grpctest` eliminates
this by providing an in-memory server with a simple API inspired by
[`httptest`](https://pkg.go.dev/net/http/httptest).

## Prerequisites

You will need the following things properly installed on your computer:

- [Go](https://golang.org/): any one of the **three latest major**
  [releases](https://golang.org/doc/devel/release.html)

## Installation

With [Go module](https://go.dev/wiki/Modules) support (Go 1.11+), simply add the
following import

```go
import "github.com/tomasbasham/grpctest"
```

to your code, and then `go [build|run|test]` will automatically fetch the
necessary dependencies.

Otherwise, to install the `grpctest` module, run the following command:

```bash
go get -u github.com/tomasbasham/grpctest
```

## Philosophy

This module uses `bufconn.Listener` for in-memory connections rather than real
network sockets. This makes tests:

- **Fast**: 10-50x faster without kernel and network stack overhead
- **Reliable**: No port conflicts or network timing issues
- **Simple**: No network resource management

TLS is intentionally omitted. Modern architectures terminate TLS at
infrastructure boundaries (load balancers, service meshes), and services
communicate over plaintext within the trust boundary. Tests should verify
service behaviour, not transport encryption. If you need TLS for specific
scenarios (e.g., mutual TLS authentication), configure it via
`grpc.ServerOption`.

The API mirrors `grpc.NewServer()` for immediate familiarity, and services must
be registered before calling `Serve()` to match standard gRPC usage patterns.

## Usage

To use this module, create a test server using the provided constructor,
register your gRPC service implementation, and establish client connections
through the test server's buffered connection. This approach allows you to test
your service handlers against real gRPC transport mechanics whilst maintaining
test isolation and performance.

### Basic Server Testing

Start by implementing your gRPC service. This example shows a simple echo
service:

<!-- pyml disable md013 -->
```go
package server

import (
    "context"

    echopb "github.com/tomasbasham/grpctest/echo/v1"
)

type EchoServer struct {
    echopb.UnimplementedEchoServiceServer
}

func (s *EchoServer) Echo(ctx context.Context, in *echopb.EchoRequest) (*echopb.EchoResponse, error) {
    return &echopb.EchoResponse{Message: in.GetMessage()}, nil
}
```
<!-- pyml enable md013 -->

In your tests, use `grpctest.NewServer()` to create a test server that starts
immediately. The server manages its own buffered connection and exposes a
`ClientConn()` method to create properly configured client connections:

<!-- pyml disable md013 -->
```go
package server_test

import (
    "context"
    "testing"

    "github.com/tomasbasham/echo/internal/server"
    "github.com/tomasbasham/grpctest"

    echopb "github.com/tomasbasham/grpctest/echo/v1"
)

func TestEcho(t *testing.T) {
    s := grpctest.NewServer()
    s.CloseOnCleanup(t)

    // Register Echo service.
    echopb.RegisterEchoServiceServer(s, &server{})
    s.Serve()

    client, err := s.ClientConn()
    if err != nil {
        t.Fatalf("failed to create client: %v", err)
   }

    // The message to echo.
    want := "Hello, world"

    echoClient := echopb.NewEchoServiceClient(client)
    resp, err := echoClient.Echo(context.Background(), &echopb.EchoRequest{Message: want})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    got := resp.GetMessage()
    if got != want {
        t.Errorf("mismatch:\n  got:  %q\n  want: %q", got, want)
    }
}
```
<!-- pyml enable md013 -->

### Advanced Configuration

For scenarios requiring custom server or client options, the testing server can
be configured just like a regular gRPC server.

```go
func TestEcho(t *testing.T) {
    // Create server with custom options.
    s := grpctest.NewServer(
        grpc.MaxRecvMsgSize(1024 * 1024 * 10),
    )
    s.CloseOnCleanup(t)

    // Register Echo service before starting.
    echopb.RegisterEchoServiceServer(s, &server.EchoServer{})
    s.Serve()

    conn, err := s.ClientConn()
    if err != nil {
        t.Fatalf("failed to create client: %v", err)
    }

    // Same as above...
}
```

## License

This project is licensed under the [MIT License](LICENSE).
