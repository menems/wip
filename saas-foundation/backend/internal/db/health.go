package db

import (
	"context"
	"encoding/json"
	"net/http"
)

// Pinger is satisfied by any type that can verify DB connectivity.
// *pgxpool.Pool implements this interface.
type Pinger interface {
	Ping(ctx context.Context) error
}

// healthResponse is the JSON shape returned by the health endpoint.
type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

// HealthHandler returns an http.HandlerFunc that responds to liveness checks.
// It pings the database via the Pinger interface and reports its status.
func HealthHandler(pinger Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := healthResponse{Status: "ok", DB: "ok"}

		if err := pinger.Ping(r.Context()); err != nil {
			resp.DB = "error"
			resp.Status = "degraded"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(resp) //nolint:errcheck
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}
