package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/blaz/serve/platform/server"
)

func startServer(t *testing.T) (*server.Server, string, context.CancelFunc) {
	t.Helper()
	cfg := server.DefaultConfig()
	cfg.Port = 0
	cfg.ShutdownTimeout = 2 * time.Second

	srv := server.New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()

	addr := srv.Addr()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("server shutdown error: %v", err)
		}
	})
	return srv, addr, cancel
}

func TestServerHealthEndpoint(t *testing.T) {
	_, addr, _ := startServer(t)

	resp, err := http.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("want status=ok, got %q", body["status"])
	}
	if _, ok := body["uptime"]; !ok {
		t.Error("missing uptime field")
	}
}

func TestServerEchoEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{"json body", `{"msg":"hello"}`, "application/json", http.StatusOK},
		{"plain text", "ping", "text/plain", http.StatusOK},
	}

	_, addr, _ := startServer(t)
	client := &http.Client{Timeout: 5 * time.Second}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "http://"+addr+"/echo", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("POST /echo: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("want %d, got %d", tc.wantStatus, resp.StatusCode)
			}
			got, _ := io.ReadAll(resp.Body)
			if string(got) != tc.body {
				t.Errorf("echo body mismatch: want %q, got %q", tc.body, string(got))
			}
			if ct := resp.Header.Get("Content-Type"); ct != tc.contentType {
				t.Errorf("Content-Type mismatch: want %q, got %q", tc.contentType, ct)
			}
		})
	}
}

func TestServerNotFound(t *testing.T) {
	_, addr, _ := startServer(t)

	resp, err := http.Get("http://" + addr + "/nonexistent")
	if err != nil {
		t.Fatalf("GET /nonexistent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if body["error"] != "not found" {
		t.Errorf("want error=not found, got %q", body["error"])
	}
	if body["path"] != "/nonexistent" {
		t.Errorf("want path=/nonexistent, got %q", body["path"])
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	cfg := server.DefaultConfig()
	cfg.Port = 0
	cfg.ShutdownTimeout = 2 * time.Second

	var out bytes.Buffer
	srv := server.New(cfg, slog.New(slog.NewTextHandler(&out, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()

	addr := srv.Addr()
	resp, err := http.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("server not up: %v", err)
	}
	resp.Body.Close()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within timeout")
	}

	if !strings.Contains(out.String(), "shutdown complete") {
		t.Errorf("expected shutdown message, got: %q", out.String())
	}
}
