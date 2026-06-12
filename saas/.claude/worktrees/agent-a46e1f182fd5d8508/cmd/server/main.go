package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"connectrpc.com/connect"
	"github.com/menems/saas/gen/api/v1/pb/pbconnect"
	"github.com/menems/saas/internal/auth"
	"github.com/menems/saas/internal/user"
	userdb "github.com/menems/saas/internal/user/db"
	"github.com/menems/saas/pkg/authz"
	gocrypt "github.com/menems/saas/pkg/bcrypt"
	"github.com/menems/saas/pkg/health"
	"github.com/menems/saas/pkg/logging"
	"github.com/menems/saas/pkg/metrics"
	"github.com/menems/saas/pkg/postgres"
	"github.com/menems/saas/pkg/server"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	// --- Config ---
	dbURL, err := requireEnv("DATABASE_URL")
	if err != nil {
		return err
	}
	jwtSecretStr, err := requireEnv("JWT_SECRET")
	if err != nil {
		return err
	}
	jwtSecret := []byte(jwtSecretStr)

	// --- Database ---
	pool, err := postgres.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	queries := userdb.New(pool.PGX())

	// --- Dependencies ---
	hasher := gocrypt.New(gocrypt.DefaultCost)
	repo := user.NewRepository(queries)
	userSvc := user.NewService(repo, hasher)
	authSvc := auth.NewService(repo, hasher, jwtSecret, 24*time.Hour)

	// --- Metrics ---
	reg := prometheus.NewRegistry()
	metricsInterceptor := metrics.NewInterceptor(reg)

	// --- Interceptors ---
	authInterceptor := authz.NewInterceptor(
		authSvc,
		authz.WithPublicProcedure(pbconnect.AuthServiceLoginProcedure),
	)
	logInterceptor := logging.NewInterceptor(log)

	// --- Server ---
	registrars := []server.RouteRegistrar{
		health.New(pool),
		metrics.NewHandler(reg),
		auth.NewHandler(authSvc),
		user.NewHandler(userSvc),
	}

	srvOpts := []server.Option{
		server.WithLogger(log),
		server.WithConnectOptions(connect.WithInterceptors(metricsInterceptor, logInterceptor, authInterceptor)),
	}
	if addr := os.Getenv("ADDR"); addr != "" {
		srvOpts = append(srvOpts, server.WithAddr(addr))
	}

	if err := server.New(registrars, srvOpts...).Run(ctx); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable not set: %s", key)
	}
	return v, nil
}
