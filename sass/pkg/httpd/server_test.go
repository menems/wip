package httpd_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/menems/sass/pkg/httpd"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestHandleHealth(t *testing.T) {
	t.Parallel()

	srv := httpd.New(":0", httpd.WithLogger(discardLogger))
	r := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, r)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("body = %q, want %q", string(body), `{"status":"ok"}`)
	}
}

func TestHandleRoot(t *testing.T) {
	t.Parallel()

	srv := httpd.New(":0", httpd.WithLogger(discardLogger))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, r)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/plain" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/plain")
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Hello, world!\n" {
		t.Errorf("body = %q, want %q", string(body), "Hello, world!\n")
	}
}

func TestRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"health", http.MethodGet, "/health/live", http.StatusOK},
		{"root", http.MethodGet, "/", http.StatusOK},
		{"method not allowed on health", http.MethodPost, "/health/live", http.StatusMethodNotAllowed},
	}

	srv := httpd.New(":0", httpd.WithLogger(discardLogger))
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(context.Background(), tt.method, ts.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			t.Cleanup(func() { _ = resp.Body.Close() })

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestRun_gracefulShutdown(t *testing.T) {
	t.Parallel()

	srv := httpd.New(":0", httpd.WithLogger(discardLogger))
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- srv.Run(ctx)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("server did not shut down within 5s")
	}
}

func TestNew_defaults(t *testing.T) {
	t.Parallel()

	srv := httpd.New(":0")

	r := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
