// Package grpctest provides a simple in-memory gRPC server for testing
// purposes.
//
// This package enables fast, isolated integration tests of gRPC services
// without requiring network sockets, port allocation, or complex setup. It
// mirrors the design philosophy of [httptest.Server] whilst respecting the
// unique characteristics of gRPC.
package grpctest
