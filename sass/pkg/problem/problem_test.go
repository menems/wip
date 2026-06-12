package problem_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/menems/sass/pkg/problem"
)

func TestWriteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		requestID     string
		status        int
		title         string
		detail        string
		wantRequestID string
	}{
		{
			name:          "happy path with request id",
			requestID:     "abc-123",
			status:        http.StatusBadRequest,
			title:         "Bad Request",
			detail:        "field 'name' is required",
			wantRequestID: "abc-123",
		},
		{
			name:          "missing X-Request-Id omitted from body",
			requestID:     "",
			status:        http.StatusBadRequest,
			title:         "Bad Request",
			detail:        "some detail",
			wantRequestID: "",
		},
		{
			name:          "404 not found",
			requestID:     "req-404",
			status:        http.StatusNotFound,
			title:         "Not Found",
			detail:        "resource /foo does not exist",
			wantRequestID: "req-404",
		},
		{
			name:          "500 internal server error",
			requestID:     "req-500",
			status:        http.StatusInternalServerError,
			title:         "Internal Server Error",
			detail:        "unexpected nil pointer",
			wantRequestID: "req-500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.requestID != "" {
				r.Header.Set("X-Request-Id", tt.requestID)
			}
			w := httptest.NewRecorder()

			problem.WriteError(w, r, tt.status, tt.title, tt.detail)

			resp := w.Result()
			t.Cleanup(func() { _ = resp.Body.Close() })

			// Status code
			if resp.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.status)
			}

			// Content-Type
			if ct := resp.Header.Get("Content-Type"); ct != "application/problem+json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/problem+json")
			}

			// Body — valid JSON
			var p problem.Problem
			if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
				t.Fatalf("decode body: %v", err)
			}

			// RFC 7807 required fields
			if p.Type == "" {
				t.Error("type is empty")
			}
			if p.Title != tt.title {
				t.Errorf("title = %q, want %q", p.Title, tt.title)
			}
			if p.Status != tt.status {
				t.Errorf("status = %d, want %d", p.Status, tt.status)
			}
			if p.Detail != tt.detail {
				t.Errorf("detail = %q, want %q", p.Detail, tt.detail)
			}
			if p.RequestID != tt.wantRequestID {
				t.Errorf("request_id = %q, want %q", p.RequestID, tt.wantRequestID)
			}
		})
	}
}
