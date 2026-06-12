package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// base valid environment
	base := map[string]string{
		"DATABASE_URL": "postgres://u:p@localhost/db",
		"JWT_SECRET":   "a-secret-that-is-at-least-32-characters",
		"CORS_ORIGIN":  "http://localhost:5173",
	}

	setEnv := func(t *testing.T, extra map[string]string) func() {
		t.Helper()
		all := make(map[string]string)
		for k, v := range base {
			all[k] = v
		}
		for k, v := range extra {
			all[k] = v
		}
		for k, v := range all {
			t.Setenv(k, v)
		}
		return func() {}
	}

	tests := []struct {
		name      string
		env       map[string]string
		wantErr   bool
		errSubstr string
		check     func(t *testing.T, cfg *Config)
	}{
		{
			name: "loads defaults when optional vars absent",
			env:  map[string]string{},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 15*time.Minute, cfg.JWTAccessTTL)
				assert.Equal(t, 720*time.Hour, cfg.JWTRefreshTTL)
				assert.Equal(t, 8080, cfg.Port)
			},
		},
		{
			name: "parses custom PORT",
			env:  map[string]string{"PORT": "9090"},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 9090, cfg.Port)
			},
		},
		{
			name: "parses custom JWT TTLs",
			env:  map[string]string{"JWT_ACCESS_TTL": "30m", "JWT_REFRESH_TTL": "48h"},
			check: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 30*time.Minute, cfg.JWTAccessTTL)
				assert.Equal(t, 48*time.Hour, cfg.JWTRefreshTTL)
			},
		},
		{
			name:      "errors when DATABASE_URL missing",
			env:       map[string]string{"DATABASE_URL": ""},
			wantErr:   true,
			errSubstr: "DATABASE_URL",
		},
		{
			name:      "errors when JWT_SECRET missing",
			env:       map[string]string{"JWT_SECRET": ""},
			wantErr:   true,
			errSubstr: "JWT_SECRET",
		},
		{
			name:      "errors when JWT_SECRET too short",
			env:       map[string]string{"JWT_SECRET": "tooshort"},
			wantErr:   true,
			errSubstr: "JWT_SECRET must be at least 32",
		},
		{
			name:      "errors when CORS_ORIGIN missing",
			env:       map[string]string{"CORS_ORIGIN": ""},
			wantErr:   true,
			errSubstr: "CORS_ORIGIN",
		},
		{
			name:      "errors on invalid PORT",
			env:       map[string]string{"PORT": "notanumber"},
			wantErr:   true,
			errSubstr: "PORT",
		},
		{
			name:      "errors on invalid JWT_ACCESS_TTL",
			env:       map[string]string{"JWT_ACCESS_TTL": "bad"},
			wantErr:   true,
			errSubstr: "JWT_ACCESS_TTL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.env)

			cfg, err := Load()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
