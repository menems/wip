// Package problem implements RFC 7807 "Problem Details for HTTP APIs".
package problem

import (
	"encoding/json"
	"net/http"
)

// Problem is an RFC 7807 problem details object.
type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteError writes an application/problem+json response with the given HTTP
// status code, title, and detail message. The RequestID field is populated from
// the X-Request-Id request header when present.
func WriteError(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	p := Problem{
		Type:      "about:blank",
		Title:     title,
		Status:    status,
		Detail:    detail,
		RequestID: r.Header.Get("X-Request-Id"),
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
