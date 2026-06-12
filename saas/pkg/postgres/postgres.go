package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps a pgxpool.Pool with a functional-options constructor and helpers.
type Pool struct {
	pool     *pgxpool.Pool
	maxConns int32
}

// Option configures a Pool.
type Option func(*Pool)

// WithMaxConns sets the maximum number of connections in the pool (default 10).
func WithMaxConns(n int32) Option {
	return func(p *Pool) {
		p.maxConns = n
	}
}

// New creates and opens a connection pool for the given DSN.
// It does NOT ping; call Ping separately for a liveness check.
func New(ctx context.Context, dsn string, opts ...Option) (*Pool, error) {
	p := &Pool{
		maxConns: 10,
	}
	for _, opt := range opts {
		opt(p)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	cfg.MaxConns = p.maxConns

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: open pool: %w", err)
	}
	p.pool = pool
	return p, nil
}

// Ping validates that the pool can reach the database.
func (p *Pool) Ping(ctx context.Context) error {
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres: ping: %w", err)
	}
	return nil
}

// PGX returns the underlying *pgxpool.Pool for use with SQLC-generated queriers.
func (p *Pool) PGX() *pgxpool.Pool {
	return p.pool
}

// Close closes the pool and releases all connections.
func (p *Pool) Close() {
	p.pool.Close()
}
