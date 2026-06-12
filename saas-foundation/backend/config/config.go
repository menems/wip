// Package config loads application configuration from environment variables.
// All required variables cause the process to exit with a descriptive message
// if they are missing or invalid.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the server.
type Config struct {
	// DatabaseURL is the PostgreSQL DSN (e.g. postgres://user:pass@host/db).
	DatabaseURL string

	// JWTSecret is the HS256 signing key. Must be at least 32 characters.
	JWTSecret string

	// JWTAccessTTL is how long access tokens remain valid (default 15m).
	JWTAccessTTL time.Duration

	// JWTRefreshTTL is how long refresh tokens remain valid (default 720h).
	JWTRefreshTTL time.Duration

	// CORSOrigin is the allowed frontend origin for CORS headers.
	CORSOrigin string

	// Port is the TCP port the HTTP server listens on (default 8080).
	Port int

	// OTELEndpoint is the base URL of the OTLP/HTTP receiver
	// (e.g. "http://localhost:4318"). When empty, OpenTelemetry is disabled
	// and the global SDK no-op providers are used. Sourced from the standard
	// OTEL_EXPORTER_OTLP_ENDPOINT environment variable.
	OTELEndpoint string

	// OTELServiceName is the service.name resource attribute sent to the
	// collector. Defaults to "saas-foundation". Sourced from OTEL_SERVICE_NAME.
	OTELServiceName string
}

// Load reads environment variables and returns a Config.
// Returns an error describing the first missing or invalid variable.
func Load() (*Config, error) {
	cfg := &Config{}

	var err error

	cfg.DatabaseURL, err = requireEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	cfg.JWTSecret, err = requireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("config: JWT_SECRET must be at least 32 characters")
	}

	cfg.JWTAccessTTL, err = parseDuration("JWT_ACCESS_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}

	cfg.JWTRefreshTTL, err = parseDuration("JWT_REFRESH_TTL", 720*time.Hour)
	if err != nil {
		return nil, err
	}

	cfg.CORSOrigin, err = requireEnv("CORS_ORIGIN")
	if err != nil {
		return nil, err
	}

	cfg.Port, err = parseInt("PORT", 8080)
	if err != nil {
		return nil, err
	}

	// OTel fields are optional; missing values are accepted as empty/default.
	cfg.OTELEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	cfg.OTELServiceName = os.Getenv("OTEL_SERVICE_NAME")
	if cfg.OTELServiceName == "" {
		cfg.OTELServiceName = "saas-foundation"
	}

	return cfg, nil
}

// requireEnv returns the value of an environment variable or an error if it is
// not set or empty.
func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("config: required environment variable %s is not set", key)
	}
	return v, nil
}

// parseDuration reads a Go duration string from an environment variable.
// Returns the defaultValue if the variable is not set.
func parseDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s is not a valid duration: %w", key, err)
	}
	return d, nil
}

// parseInt reads an integer from an environment variable.
// Returns the defaultValue if the variable is not set.
func parseInt(key string, defaultValue int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s is not a valid integer: %w", key, err)
	}
	return n, nil
}
