// Package db manages the database connection pool.
// Schema migrations are handled externally via the migrate CLI:
//
//	migrate -path internal/db/migrations -database "$DATABASE_URL" up
package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is a convenience alias so callers only need to import this package.
type Pool = pgxpool.Pool

// NewPool opens and pings a connection pool. It does not run migrations.
func NewPool(ctx context.Context) (*Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
