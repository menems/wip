package httpd

import (
	"context"
	"net/http"

	"golang.org/x/sync/errgroup"

	"github.com/menems/sass/pkg/problem"
)

// Checker is implemented by any dependency that can report its health.
type Checker interface {
	Check(ctx context.Context) error
}

func (s *Server) handleLive() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}

func (s *Server) handleReady(checkers []Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g, ctx := errgroup.WithContext(r.Context())
		for _, c := range checkers {
			g.Go(func() error {
				return c.Check(ctx)
			})
		}
		if err := g.Wait(); err != nil {
			problem.WriteError(w, r, http.StatusServiceUnavailable, "Service Unavailable", err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
