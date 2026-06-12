package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blaz/serve/internal/contacts"
	"github.com/blaz/serve/internal/users"
	"github.com/blaz/serve/platform/auth"
	"github.com/blaz/serve/platform/postgres"
	"github.com/blaz/serve/platform/server"
)

func main() {
	os.Exit(run())
}

func run() int {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := server.DefaultConfig()

	flag.StringVar(&cfg.Host, "host", cfg.Host, "host address to listen on")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "port to listen on")

	var shutdownTimeout time.Duration
	flag.DurationVar(&shutdownTimeout, "shutdown-timeout", cfg.ShutdownTimeout,
		"graceful shutdown timeout (e.g. 5s, 10s)")

	var dsn string
	flag.StringVar(&dsn, "dsn", "", "postgres DSN (omit to use in-memory store)")

	flag.Parse()

	cfg.ShutdownTimeout = shutdownTimeout

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var (
		contactRepo contacts.Repo
		userRepo    users.Repo
	)

	if dsn != "" {
		pool, err := postgres.New(ctx, dsn)
		if err != nil {
			log.Error("postgres connect", "err", err)
			return 1
		}
		defer pool.Close()
		contactRepo = contacts.NewPGRepository(pool)
		userRepo = users.NewPGRepository(pool)
	} else {
		contactRepo = contacts.NewRepository()
		userRepo = users.NewRepository()
	}

	userSvc := users.NewService(userRepo)
	contactSvc := contacts.NewService(contactRepo)

	authInterceptor := auth.NewInterceptor(userSvc)

	srv := server.New(cfg, log,
		server.WithRoutes(users.NewConnectHandler(userSvc)),
		server.WithRoutes(contacts.NewConnectHandler(contactSvc, authInterceptor)),
	)
	if err := srv.Run(ctx); err != nil {
		log.Error("serve", "err", err)
		return 1
	}
	return 0
}
