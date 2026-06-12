# CLAUDE.md

## Commands

### Full stack (from repo root)
```bash
make dev        # postgres + backend (air hot-reload) + frontend (Vite HMR)
make migrate    # run pending SQL migrations
make seed       # insert default admin role + user (idempotent)
make build      # build all Docker images
```

### Backend (from `backend/`)
```bash
go test ./...                                   # all tests
go test ./internal/<pkg>/... -v -race -count=1  # single package
go build ./...                                  # compile check
```

### Frontend (from `frontend/`)
```bash
npm run test:run   # all tests, CI mode
npm run test       # watch mode
npx tsc -b         # type-check only
```

Go module: `github.com/your-org/saas-foundation/backend`
Seed credentials: `admin@example.com` / `changeme`

---

@docs/backend.md
@docs/frontend.md
@docs/rbac.md
