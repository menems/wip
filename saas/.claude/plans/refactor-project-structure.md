# refactor-project-structure
> Align backend layout with `.claude/idioms/go.md` conventions

**Created**: 2026-03-12 | **Branch**: feat/refactor-project-structure

## Steps

1. [backend] Move proto files to `proto/api/v1/` and update buf config
   → `buf generate` succeeds; imports compile; no `.proto` files remain under `internal/`

2. [backend] Move migrations from `sql/schema/` to `db/migrations/` and update Makefile
   → `make migrate-up` and `make migrate-down` work against the new path

3. [backend] Move queries per-feature and reconfigure SQLC with `emit_interface: true`
   → `sqlc generate` produces `internal/user/db/` with a `Querier` interface; shared `internal/db/` is deleted

4. [backend] Add sentinel errors to each feature's domain file
   → Domain files export `ErrNotFound`, `ErrConflict`, `ErrValidation`; service methods wrap them

5. [backend] Rename `UserStore` → `Store`; move service interfaces into `handler.go` as local interfaces
   → Interfaces are locally defined where consumed; no global service interfaces in domain files

6. [backend] Split service methods into `usecase_*.go` files
   → Each use case is a separate file; `service.go` has only `Service` struct, `Store` interface, and `NewService`

7. [backend] Add single error-mapping helper in each `handler.go`; remove ad-hoc `connect.NewError`
   → Zero direct `connect.NewError` calls outside the mapping helper; sentinel → code mapping is centralized

8. [backend] Introduce `RouteRegistrar` pattern; refactor `main.go`
   → `main.go` iterates registrars; adding a new feature requires no changes to `main.go`

9. [backend] Verify layer boundaries: service never imports `db`, handler never imports storage
   → `go vet ./...` passes; no cross-layer imports

