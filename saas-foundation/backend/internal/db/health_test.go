package db

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPinger implements Pinger for unit tests.
type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(_ context.Context) error { return m.err }

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		pingerErr  error
		wantStatus int
		wantBody   healthResponse
	}{
		{
			name:       "returns 200 ok when DB is healthy",
			pingerErr:  nil,
			wantStatus: http.StatusOK,
			wantBody:   healthResponse{Status: "ok", DB: "ok"},
		},
		{
			name:       "returns 503 degraded when DB ping fails",
			pingerErr:  errors.New("connection refused"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   healthResponse{Status: "degraded", DB: "error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := HealthHandler(&mockPinger{err: tt.pingerErr})

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			handler(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var body healthResponse
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
			assert.Equal(t, tt.wantBody, body)
		})
	}
}
