package semantic

import (
	"testing"
)

// TestClose_NilConn covers the conn == nil path in Close.
func TestClose_NilConn(t *testing.T) {
	vs := NewWithClients(nil, nil, "test")
	err := vs.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestNew_InvalidAddr tests New with an addr that can connect (grpc.NewClient doesn't
// actually dial by default in newer gRPC versions). This covers the New function.
func TestNew_Success(t *testing.T) {
	vs, err := New("localhost:0", "test-collection")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vs == nil {
		t.Fatal("expected non-nil store")
	}
	// Clean up
	vs.Close()
}
