package server_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/menems/saas/pkg/server"
)

// noopRegistrar is a RouteRegistrar that mounts nothing.
type noopRegistrar struct{}

func (noopRegistrar) Register(_ *http.ServeMux, _ ...connect.HandlerOption) {}

func TestServer_StartStop(t *testing.T) {
	t.Parallel()

	s := server.New(
		[]server.RouteRegistrar{noopRegistrar{}},
		server.WithAddr("127.0.0.1:19999"),
		server.WithReadHeaderTimeout(time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- s.Run(ctx)
	}()

	// Give the server a moment to start.
	time.Sleep(50 * time.Millisecond)

	// Cancel context to trigger graceful shutdown.
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}
