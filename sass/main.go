// Package main is the entrypoint for the sass server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/menems/sass/internal/auth"
	"github.com/menems/sass/internal/roles"
	"github.com/menems/sass/internal/users"
	"github.com/menems/sass/pkg/httpd"
	"github.com/menems/sass/pkg/storage/postgres"
	"github.com/menems/sass/pkg/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := run(logger); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	svcName := os.Getenv("OTEL_SERVICE_NAME")
	if svcName == "" {
		svcName = "sass"
	}
	tp, err := telemetry.NewTracerProvider(ctx, svcName)
	if err != nil {
		return fmt.Errorf("tracer provider: %w", err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	pool, err := postgres.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer pool.Close()

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	tokenCfg := auth.TokenConfig{
		Secret:     jwtSecret,
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 720 * time.Hour,
	}

	authRepo := auth.NewDBRepository(pool)
	authSvc := auth.NewService(authRepo, authRepo, tokenCfg)
	authH := auth.NewHandler(authSvc, authSvc, os.Getenv("ENV") == "production")

	usersRepo := users.NewDBRepository(pool)
	usersSvc := users.NewUserService(usersRepo, usersRepo, usersRepo)
	usersH := users.NewHandler(usersSvc, usersSvc, usersSvc)

	rolesRepo := roles.NewDBRepository(pool)
	rolesSvc := roles.NewService(rolesRepo, rolesRepo, rolesRepo)
	rolesH := roles.NewHandler(rolesSvc, rolesSvc)

	jwtMW := auth.JWTMiddleware(authSvc)
	requirePerm := auth.RequirePerm(authSvc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)

	srv := httpd.New(addr,
		httpd.WithLogger(logger),
		httpd.WithTracerProvider(tp),
		httpd.WithRoutes(func(r chi.Router) {
			authH.Mount(r)
			r.Group(func(r chi.Router) {
				r.Use(jwtMW)
				r.Get("/api/v1/auth/me", authH.Me())
				usersH.Mount(r, requirePerm)
				rolesH.Mount(r, requirePerm)
			})
		}),
	)

	logger.Info("starting server", slog.String("addr", addr))

	if err := srv.Run(ctx); err != nil {
		return fmt.Errorf("server: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
