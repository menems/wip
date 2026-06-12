package server

import "net/http"

// Option configures a Server.
type Option func(*Server)

// RouteRegistrar can mount its routes onto an http.ServeMux.
type RouteRegistrar interface {
	RegisterRoutes(*http.ServeMux)
}

// WithRoutes registers one or more RouteRegistrars on the server.
func WithRoutes(rr ...RouteRegistrar) Option {
	return func(s *Server) {
		s.registrars = append(s.registrars, rr...)
	}
}
