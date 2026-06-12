# saas

SaaS starter — JWT authentication and user management via ConnectRPC.

## Development

### Prerequisites

- Go 1.26+
- Docker (for local Postgres via `make up`)
- `buf` — managed via `go tool buf`

### Run

```bash
make up        # start Postgres
make migrate   # apply migrations
make run       # start the server
```

### Seed

A default admin user (`admin@localhost` / `admin1234`) is inserted by migration `000002`.

To generate a new bcrypt hash (e.g. to rotate the seed password):

```bash
go run ./cmd/gen-bcrypt-hash <password>
```

Copy the output into `sql/schema/000002_seed_admin.up.sql`.

### Generate protobuf/sqlc

```bash
make generate
```
