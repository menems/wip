// Command server is the entry point for the saas-foundation API server.
// It wires configuration, DB connection, migrations, and the HTTP router.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/your-org/saas-foundation/backend/config"
	"github.com/your-org/saas-foundation/backend/internal/audit"
	"github.com/your-org/saas-foundation/backend/internal/auth"
	"github.com/your-org/saas-foundation/backend/internal/db"
	appmiddleware "github.com/your-org/saas-foundation/backend/internal/middleware"
	"github.com/your-org/saas-foundation/backend/internal/roles"
	"github.com/your-org/saas-foundation/backend/internal/telemetry"
	"github.com/your-org/saas-foundation/backend/internal/users"
)

func main() {
	ctx := context.Background()

	// -------------------------------------------------------------------------
	// Configuration
	// -------------------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// -------------------------------------------------------------------------
	// OpenTelemetry — must be set up before any other component so that the
	// global TracerProvider and MeterProvider are in place when the HTTP
	// middleware is constructed. No-op when OTEL_EXPORTER_OTLP_ENDPOINT is empty.
	// -------------------------------------------------------------------------
	otelShutdown, err := telemetry.Setup(ctx, cfg.OTELServiceName, cfg.OTELEndpoint)
	if err != nil {
		log.Fatalf("telemetry: %v", err)
	}
	defer func() {
		if shutdownErr := otelShutdown(ctx); shutdownErr != nil {
			log.Printf("telemetry: shutdown: %v", shutdownErr)
		}
	}()

	// -------------------------------------------------------------------------
	// Sub-commands (migrate, seed)
	// -------------------------------------------------------------------------
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			runMigrations(ctx, cfg)
			return
		case "seed":
			log.Println("seed: seed data is applied via migration 002_seed.sql — run 'migrate' instead")
			return
		default:
			log.Fatalf("unknown sub-command: %s", os.Args[1])
		}
	}

	// -------------------------------------------------------------------------
	// Database
	// -------------------------------------------------------------------------
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	log.Println("database: connected")

	// -------------------------------------------------------------------------
	// Router
	// -------------------------------------------------------------------------
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(appmiddleware.OTel()) // traces + http.server.request.duration metric
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS — simple manual header injection for now;
	// full per-origin CORS is wired in later tasks.
	r.Use(corsMiddleware(cfg.CORSOrigin))

	// Health (unauthenticated) — pool satisfies db.Pinger directly
	r.Get("/health", db.HealthHandler(pool))

	// Auth service + handler
	authRepo := auth.NewDBRepository(pool)
	authSvc := auth.NewService(authRepo, auth.TokenConfig{
		Secret:     []byte(cfg.JWTSecret),
		AccessTTL:  cfg.JWTAccessTTL,
		RefreshTTL: cfg.JWTRefreshTTL,
	})
	authHandler := auth.NewHandler(authSvc, true /* secure cookies in production */)

	// Permission loader for RBAC middleware
	permLoader := appmiddleware.NewDBPermissionLoader(pool)

	// API v1 group
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (no JWT required)
		authHandler.Mount(r)

		// All routes below require a valid access token
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.JWTAuth(authSvc))

			// /auth/me
			r.Get("/auth/me", authHandler.Me())

			// requirePerm is a helper that wires RequirePermission with our loader.
			requirePerm := func(resource, action string) func(http.Handler) http.Handler {
				return appmiddleware.RequirePermission(permLoader, resource, action)
			}

			// Users API
			usersRepo := users.NewDBRepository(pool)
			usersSvc := users.NewService(usersRepo, 12 /* bcrypt cost */)
			usersHandler := users.NewHandler(usersSvc)
			usersHandler.Mount(r, requirePerm)

			// Roles API
			rolesRepo := roles.NewDBRepository(pool)
			rolesSvc := roles.NewService(rolesRepo)
			rolesHandler := roles.NewHandler(rolesSvc)
			rolesHandler.Mount(r, requirePerm)

			// Audit Logs API
			auditRepo := audit.NewDBRepository(pool)
			auditSvc := audit.NewService(auditRepo)
			auditHandler := audit.NewHandler(auditSvc)
			auditHandler.Mount(r, requirePerm)
		})
	})

	// -------------------------------------------------------------------------
	// HTTP Server
	// -------------------------------------------------------------------------
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("server: listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

// runMigrations applies all pending SQL migrations and exits.
func runMigrations(ctx context.Context, cfg *config.Config) {
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("migrate: db connect: %v", err)
	}
	defer pool.Close()

	migrationsDir := migrationsPath()
	log.Printf("migrate: applying migrations from %s", migrationsDir)

	if err := db.Migrate(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrate: done")
}

// migrationsPath resolves the migrations directory relative to the binary or
// the source root (convenient for both production and local development).
func migrationsPath() string {
	// When running via `go run ./cmd/server`, the source root is the module root.
	// In production the binary sits next to the migrations directory.
	_, filename, _, _ := runtime.Caller(0)
	// filename = .../cmd/server/main.go → go up 3 levels to backend/
	root := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
	if _, err := os.Stat(root); err == nil {
		return root
	}
	// Fallback: expect migrations/ beside the binary
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "migrations")
}

// corsMiddleware adds basic CORS headers for the configured origin.
// A more complete CORS implementation (preflight, credentials) is added in
// task 1.5 alongside the auth middleware.
func corsMiddleware(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
