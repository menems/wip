package server

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const maxBodySize = 1 << 20 // 1 MiB

func (s *Server) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(s.start).Truncate(time.Second)
		s.writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"uptime": uptime.String(),
		})
	}
}

func (s *Server) echoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, `{"error":"request body too large"}`, http.StatusRequestEntityTooLarge)
			return
		}
		ct := r.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		w.Header().Set("Content-Type", ct)
		w.WriteHeader(http.StatusOK)
		w.Write(body) //nolint:errcheck
	}
}

func (s *Server) notFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not found",
			"path":  r.URL.Path,
		})
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.log.Error("writeJSON encode error", "err", err)
	}
}
