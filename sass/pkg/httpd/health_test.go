package httpd_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/menems/sass/pkg/httpd"
)

// checkerFunc is a test helper that adapts a function to the Checker interface.
type checkerFunc func(ctx context.Context) error

func (f checkerFunc) Check(ctx context.Context) error { return f(ctx) }

func TestHandleLive(t *testing.T) {
	t.Parallel()

	srv := httpd.New(":0", httpd.WithLogger(discardLogger))
	r := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, r)

	resp := w.Result()
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("body = %q, want %q", string(body), `{"status":"ok"}`)
	}
}

func TestHandleReady(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("db connection refused")

	tests := []struct {
		name       string
		checkers   []httpd.Checker
		wantStatus int
		wantBody   string
		wantCT     string
	}{
		{
			name:       "no checkers returns 200",
			checkers:   nil,
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
			wantCT:     "application/json",
		},
		{
			name: "all passing checkers returns 200",
			checkers: []httpd.Checker{
				checkerFunc(func(_ context.Context) error { return nil }),
				checkerFunc(func(_ context.Context) error { return nil }),
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
			wantCT:     "application/json",
		},
		{
			name: "failing checker returns 503 problem+json",
			checkers: []httpd.Checker{
				checkerFunc(func(_ context.Context) error { return errBoom }),
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCT:     "application/problem+json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httpd.New(":0", httpd.WithLogger(discardLogger), httpd.WithCheckers(tt.checkers...))
			r := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
			w := httptest.NewRecorder()

			srv.Handler().ServeHTTP(w, r)

			resp := w.Result()
			t.Cleanup(func() { _ = resp.Body.Close() })

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			ct := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, tt.wantCT) {
				t.Errorf("Content-Type = %q, want prefix %q", ct, tt.wantCT)
			}

			body, _ := io.ReadAll(resp.Body)
			if tt.wantBody != "" && string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", string(body), tt.wantBody)
			}

			// For error responses, verify it is valid JSON with the expected status field.
			if tt.wantStatus == http.StatusServiceUnavailable {
				var p map[string]any
				if err := json.Unmarshal(body, &p); err != nil {
					t.Fatalf("response body is not valid JSON: %v\nraw: %s", err, body)
				}
				if status, _ := p["status"].(float64); int(status) != http.StatusServiceUnavailable {
					t.Errorf("problem.status = %v, want %d", p["status"], http.StatusServiceUnavailable)
				}
				if detail, _ := p["detail"].(string); detail == "" {
					t.Error("problem.detail is empty, want error message")
				}
			}
		})
	}
}
