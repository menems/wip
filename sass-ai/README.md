# SaaS App

Go + React SaaS starter with email/password auth, JWT, PostgreSQL, and a profile page.

## Stack

- **Backend**: Go, chi, pgx, golang-jwt, bcrypt
- **Frontend**: Bun, React 18, TypeScript, Tailwind CSS, TanStack Router/Query
- **Database**: PostgreSQL 16

## Quick Start

### 1. Start PostgreSQL

```bash
docker compose up -d
```

### 2. Configure backend

```bash
cp .env.example backend/.env
# Edit backend/.env if needed
```

### 3. Run backend

```bash
cd backend
go run ./cmd/server
```

The API runs on http://localhost:8080

### 4. Run frontend

```bash
cd frontend
bun dev
```

The app runs on http://localhost:3000

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/auth/register | No | Register with email/password |
| POST | /api/auth/login | No | Login, returns JWT |
| GET | /api/users/me | Yes | Get current user profile |
| PATCH | /api/users/me | Yes | Update name and avatar URL |

## Project Structure

```
.
├── backend/          # Go API server
│   ├── cmd/server/   # Entry point
│   ├── internal/
│   │   ├── auth/     # JWT, login/register handlers, middleware
│   │   ├── user/     # User model, repository, handlers
│   │   └── db/       # PostgreSQL connection + migrations
│   └── migrations/   # SQL migration files
├── frontend/         # React SPA
│   └── src/
│       ├── api/      # API client
│       ├── pages/    # Login, Register, Profile
│       └── router.tsx
└── docker-compose.yml
```
