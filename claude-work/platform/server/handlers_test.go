package server_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blaz/serve/platform/server"
)

func newTestServer() *server.Server {
	return server.New(server.DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestHealthHandler(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("want status 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("want Content-Type application/json, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("body missing status:ok, got %q", body)
	}
	if !strings.Contains(body, `"uptime"`) {
		t.Errorf("body missing uptime field, got %q", body)
	}
}

func TestEchoHandler(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{"json body", `{"msg":"hi"}`, "application/json", http.StatusOK},
		{"plain text body", "hello world", "text/plain", http.StatusOK},
		{"empty body", "", "application/json", http.StatusOK},
		{"body too large", strings.Repeat("x", 1<<20+1), "text/plain", http.StatusRequestEntityTooLarge},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer()
			req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			res := rec.Result()
			if res.StatusCode != tc.wantStatus {
				t.Fatalf("want status %d, got %d", tc.wantStatus, res.StatusCode)
			}
			if tc.wantStatus == http.StatusOK {
				if got := rec.Body.String(); got != tc.body {
					t.Errorf("body mismatch: want %q, got %q", tc.body, got)
				}
				if ct := res.Header.Get("Content-Type"); ct != tc.contentType {
					t.Errorf("Content-Type mismatch: want %q, got %q", tc.contentType, ct)
				}
			}
		})
	}
}

func TestNotFoundHandler(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"get unknown path", http.MethodGet, "/unknown"},
		{"post unknown path", http.MethodPost, "/foo/bar"},
		{"delete root", http.MethodDelete, "/"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			res := rec.Result()
			if res.StatusCode != http.StatusNotFound {
				t.Fatalf("want 404, got %d", res.StatusCode)
			}
			body := rec.Body.String()
			if !strings.Contains(body, `"error":"not found"`) {
				t.Errorf("body missing error field, got %q", body)
			}
			if !strings.Contains(body, tc.path) {
				t.Errorf("body missing path %q, got %q", tc.path, body)
			}
		})
	}
}

func TestHealthUptime(t *testing.T) {
	srv := newTestServer()
	get := func() string {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	first := get()
	time.Sleep(10 * time.Millisecond)
	second := get()
	for _, body := range []string{first, second} {
		if !strings.Contains(body, `"uptime"`) {
			t.Errorf("missing uptime in %q", body)
		}
	}
	_ = second
}
