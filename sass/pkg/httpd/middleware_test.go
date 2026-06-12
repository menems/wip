package httpd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/menems/sass/pkg/httpd"
)

func TestSlogMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"health ok", http.MethodGet, "/health/live", http.StatusOK},
		{"root ok", http.MethodGet, "/", http.StatusOK},
		{"not found", http.MethodGet, "/missing", http.StatusNotFound},
		{"method not allowed", http.MethodPost, "/health/live", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			srv := httpd.New(":0", httpd.WithLogger(logger))

			r := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, r)

			resp := w.Result()

			// X-Request-Id must be present in the response.
			reqID := resp.Header.Get("X-Request-Id")
			if reqID == "" {
				t.Fatal("X-Request-Id header not set in response")
			}

			// Parse the single log line emitted by SlogMiddleware.
			var entry map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry); err != nil {
				t.Fatalf("parse log entry: %v\nraw: %s", err, buf.String())
			}

			// All required fields must be present.
			for _, f := range []string{"method", "path", "status", "duration_ms", "request_id"} {
				if _, ok := entry[f]; !ok {
					t.Errorf("log entry missing field %q", f)
				}
			}

			// request_id in the log must match the X-Request-Id response header.
			if got, _ := entry["request_id"].(string); got != reqID {
				t.Errorf("log request_id = %q, want %q (X-Request-Id)", got, reqID)
			}

			// status in the log must match the actual HTTP status.
			if got, _ := entry["status"].(float64); int(got) != tt.wantStatus {
				t.Errorf("log status = %v, want %d", got, tt.wantStatus)
			}
		})
	}
}
