// Package httputil provides shared HTTP response helpers.
package httputil

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ErrDetail is the canonical API error envelope body.
type ErrDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// WriteJSON encodes v as JSON and writes it with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// WriteError writes the standard error envelope JSON response.
func WriteError(w http.ResponseWriter, status int, code, message string, details any) {
	WriteJSON(w, status, map[string]any{
		"error": ErrDetail{Code: code, Message: message, Details: details},
	})
}

// ParseUUID extracts and parses a UUID path parameter by name.
// On failure it writes a 400 VALIDATION_ERROR response and returns false.
func ParseUUID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	raw := chi.URLParam(r, param)
	id, err := uuid.Parse(raw)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id format", nil)
		return uuid.Nil, false
	}
	return id, true
}
