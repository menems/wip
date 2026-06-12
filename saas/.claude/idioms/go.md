# Go Conventions

## Architecture
- Vertical slice: `internal/<feature>/` — one flat package per feature, no sub-packages
- Stack: ConnectRPC · SQLC · golang-migrate
- `pkg/` — generic technical code only; NEVER imports `internal/`

## Type Layers
| Layer | Alias | Location | Rules |
|-------|-------|----------|-------|
| Domain | — | `<pkg>.go` | plain structs, no tags |
| Proto | `pb` | `gen/api/v1/pb` | — |
| Storage | `db` | generated | — |

Convert at boundaries; never leak across layers.

## Files per `internal/<feature>/`
- `<pkg>.go` — domain structs + sentinel errors (`var ErrNotFound = …`)
- `service.go` — `Service` struct, `Store` interface, `NewService` constructor
- `handler.go` — ConnectRPC handler; imports `pb`; maps proto ↔ domain; error mapping helper
- `repository.go` — implements `Store` via `db.Querier`; maps domain ↔ storage
- `usecase_*.go` — one file per use case (methods on `Service`)
- `queries.sql` — SQLC queries

## Interfaces (always local, never global)
- `handler.go` defines `type <feature>Service interface` (only methods it calls)
- `service.go` defines `Store` interface (only methods it needs)
- Service never imports `db`; repository does

## Constructors
- `internal/`: plain constructors, explicit deps
- `pkg/`: functional options (`Option`, `WithX`) — only here

## ConnectRPC Errors
- Single mapping helper in `handler.go`: sentinel → `connect.Code`
- `ErrNotFound→CodeNotFound`, `ErrConflict→CodeAlreadyExists`, `ErrValidation→CodeInvalidArgument`
- No ad-hoc `connect.NewError` in methods
- `RouteRegistrar` pattern for registration

## Errors
- Per-feature sentinel `var`s; wrap with context: `fmt.Errorf("create user: %w", ErrNotFound)`

## Storage
- Migrations: `db/migrations/*.up.sql`
- Queries: `internal/<feature>/queries.sql`
- SQLC generates into `internal/<feature>/db/` (`emit_interface: true`)
- `pkg/postgres` — connection pool only

## Proto
- Contract-first: `proto/api/v1/*.proto` (project root)
- Generated Go code: `gen/api/v1/pb` (package `pb`) and `gen/api/v1/pb/pbconnect` (package `pbconnect`)

## Tests
- `_test` package; test exported API only
- Unit tests per layer; integration tests as final verification

## QA Gate
```bash
go test -race -count=1 ./... && go vet ./...
```

## Git invocation
- Never prepend `cd <dir>` to a git command. The worktree is already the cwd.
- If a different directory is required, use `git -C <dir> <subcommand>`.
- Rationale: `cd` before git triggers a sandbox permission prompt on every new path (e.g. each worktree), and git already operates on its working tree without it.

## Tooling
- `Makefile` targets: `generate`, `lint`, `test`, `migrate`, `run`
- Tool deps in `go.mod` `tool` section — no `tools.go`

## Creation Order
1. **Proto** — `.proto` + `buf generate` — ⛔ pause for user approval
2. **Domain** — `<pkg>.go` (structs + errors)
3. **Service** — `service.go` (`Service` struct, `Store` interface, `NewService`) + `usecase_*.go`
4. **Schema** — `db/migrations/*.up.sql`
5. **Queries** — `queries.sql` + `sqlc generate`
6. **Repository** — `repository.go`
7. **Handler** — `handler.go`
