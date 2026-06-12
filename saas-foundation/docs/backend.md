# Backend Architecture

## Package layout

Every domain package has the same four-file shape:

```
<domain>.go     ← Types, port interfaces, sentinel errors. No imports from other layers.
service.go      ← Use-case logic. Depends only on the Repository interface.
handler.go      ← HTTP adapter (chi). Translates HTTP ↔ service; owns error→status mapping.
repository.go   ← DB adapter (pgx). Implements the Repository interface.
```

**Rule:** domain ← service ← handler/repository. The domain layer never imports adapters.

`cmd/server/main.go` wiring order:
`config.Load()` → `db.Connect()` → repos → services → handlers → mount routes.

## RBAC middleware

Routes opt in per-route:
```go
r.With(mw.RequirePermission("users", "write")).Post("/", usersHandler.Create)
```
Permissions are loaded from DB per-request (not from JWT) so changes are immediate.

## Error handling

Sentinel errors declared in `<domain>.go`:
```go
var ErrNotFound      = errors.New("not found")
var ErrEmailConflict = errors.New("email already in use")
```
Handlers map with `errors.Is()` → HTTP status + JSON code (`NOT_FOUND`, `CONFLICT`, `VALIDATION_ERROR`, `FORBIDDEN`, `INTERNAL_ERROR`).

Auth deliberately conflates wrong-password and unknown-user into `ErrInvalidCredentials` (prevents email enumeration).

## DB patterns (pgx)

- Parameterized SQL with `$1 $2` (pgx style).
- Unique violations: `isUniqueViolation(err)` checks pgx error code `23505`.
- Scan via `rowScanner` interface (satisfied by both `pgx.Row` and `pgx.Rows`).
- Relationships loaded with a second query after the primary one.

## OTel middleware

`internal/middleware/OTel()` is registered globally. Creates a span per request and records `http.server.request.duration`. Env vars:
- `OTEL_EXPORTER_OTLP_ENDPOINT` — OTLP/HTTP receiver base URL. Empty = no-op.
- `OTEL_SERVICE_NAME` — default `saas-foundation`.

SDK bootstrap in `internal/telemetry/telemetry.go`.

## Testing

Table-driven tests, `testify/assert` + `testify/require`.
Service tests: hand-written mock implementing the `Repository` interface.
Handler tests: `httptest` with the mock repo wired directly.
