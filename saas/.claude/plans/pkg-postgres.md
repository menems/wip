# pkg-postgres
> add pkg/postgres managing a pgxpool connection with ping and functional options

**Created**: 2026-03-12 | **Branch**: feat/pkg-postgres

## Steps
1. [backend] Create `pkg/postgres` with Pool, functional options, and Ping
   → `New(ctx, dsn, opts...)` returns `(*Pool, error)`; functional options include `WithMaxConns`; `Pool.Ping(ctx) error` validates liveness; `Pool.PGX() *pgxpool.Pool` exposes the underlying pool for SQLC; `_test` package covers constructor defaults and option application without a live DB

2. [backend] Wire `cmd/server/main.go` to use `pkg/postgres`
   → replace inline `pgxpool.New` with `postgres.New`; call `pool.Ping` on startup to fail fast; pass `pool.PGX()` to SQLC `userdb.New`; QA gate passes (`go test -race -count=1 ./... && go vet ./...`)
