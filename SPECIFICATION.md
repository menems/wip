# SPECIFICATION.md

## 1. Overview

This project is a reusable internal SaaS foundation for bootstrapping admin dashboards and
reporting/analytics tools. It is built as a **modular monolith**: a single Go HTTP server
backed by PostgreSQL, paired with a React + Vite single-page application. The frontend and
backend communicate over REST/JSON using httpOnly cookies for stateless JWT authentication.
The entire stack runs as Docker Compose services, making it deployable anywhere without
cloud-provider coupling.

---

## 2. Architecture

```
┌─────────────────────────────────────────┐
│              Browser Client              │
│         React + Vite (SPA)              │
│   React Router · TanStack Query          │
│   shadcn/ui · Tailwind · Recharts        │
└──────────────────┬──────────────────────┘
                   │ REST/JSON  HTTPS
                   │ httpOnly cookies (access_token + refresh_token)
┌──────────────────▼──────────────────────┐
│             Go HTTP Server               │
│   ┌──────────┬──────────┬────────────┐  │
│   │   auth   │  users   │   roles    │  │
│   ├──────────┼──────────┼────────────┤  │
│   │  audit   │ middleware│  config   │  │
│   └──────────┴──────────┴────────────┘  │
│        internal/ package boundaries      │
└──────────────────┬──────────────────────┘
                   │ pgx / database/sql
┌──────────────────▼──────────────────────┐
│         PostgreSQL 16                    │
│   plain SQL migrations (numbered)        │
└─────────────────────────────────────────┘
```

- The Go backend is a **single binary** listening on `PORT` (default `8080`).
- All API routes are prefixed `/api/v1/`.
- The React SPA is served as static files by nginx in production (frontend Docker image).
- In development, Vite's dev server proxies API calls to the Go backend.
- Auth tokens travel exclusively via `SameSite=Strict; HttpOnly; Secure` cookies — never in response bodies or `Authorization` headers.

---

## 3. Repository Structure

```
saas-foundation/
├── backend/                          # Go application (single binary)
│   ├── cmd/
│   │   └── server/
│   │       └── main.go               # Entry point: wires config, DB, router
│   ├── internal/
│   │   ├── auth/                     # JWT issuance, validation, refresh logic
│   │   ├── users/                    # User CRUD, deactivation, password reset
│   │   ├── roles/                    # Role + permission management
│   │   ├── audit/                    # Audit log writes and paginated queries
│   │   ├── middleware/               # JWT auth middleware, RBAC enforcement
│   │   └── db/                       # DB connection pool, migration runner
│   ├── migrations/                   # Numbered SQL files (001_init.sql, etc.)
│   ├── config/                       # Env-based config struct + loader
│   └── Dockerfile                    # Multi-stage: build → minimal runtime image
│
├── frontend/                         # React + Vite SPA
│   ├── src/
│   │   ├── components/
│   │   │   ├── ui/                   # shadcn/ui primitives (Button, Input, Badge…)
│   │   │   ├── DataTable/            # Generic sortable/filterable/paginated table
│   │   │   ├── Charts/               # Recharts wrappers: LineChart, BarChart, PieChart
│   │   │   └── Navigation/
│   │   │       ├── Sidebar.tsx       # Collapsible, permission-aware nav links
│   │   │       ├── TopBar.tsx        # User menu + theme toggle
│   │   │       └── Breadcrumbs.tsx
│   │   ├── features/
│   │   │   ├── auth/
│   │   │   │   ├── LoginForm.tsx
│   │   │   │   ├── AuthContext.tsx   # Current user state + hasPermission() helper
│   │   │   │   └── useAuth.ts
│   │   │   ├── users/
│   │   │   │   ├── UserTable.tsx
│   │   │   │   ├── UserForm.tsx      # Shared create/edit form
│   │   │   │   └── useUsers.ts       # TanStack Query hooks
│   │   │   ├── roles/
│   │   │   │   ├── RoleTable.tsx
│   │   │   │   ├── RoleForm.tsx      # Name + permission matrix builder
│   │   │   │   └── useRoles.ts
│   │   │   └── audit/
│   │   │       ├── AuditLogTable.tsx
│   │   │       ├── AuditFilters.tsx
│   │   │       └── useAuditLogs.ts
│   │   ├── lib/
│   │   │   ├── api.ts                # Base fetch wrapper: cookies, 401→refresh, errors
│   │   │   └── utils.ts              # Shared utilities
│   │   ├── hooks/                    # Shared React hooks
│   │   ├── pages/                    # Route-level page components
│   │   ├── router.tsx                # React Router config + ProtectedRoute
│   │   ├── theme.css                 # CSS variable overrides for per-project theming
│   │   └── main.tsx                  # App entry point
│   ├── index.html
│   ├── vite.config.ts
│   └── Dockerfile                    # Multi-stage: vite build → nginx static
│
├── docker-compose.yml                # Production-like: postgres + backend + frontend
├── docker-compose.dev.yml            # Dev overrides: bind mounts, Vite HMR, air
├── Makefile                          # Targets: dev, build, migrate, test, seed
└── README.md
```

---

## 4. Technology Stack

| Layer        | Technology              | Version   | Rationale                                                  |
|--------------|-------------------------|-----------|------------------------------------------------------------|
| Frontend     | React                   | 19.x      | Required by constraints                                    |
| Build tool   | Vite                    | 6.x       | Required by constraints                                    |
| Routing      | React Router            | 7.x       | De-facto standard for React SPAs                           |
| UI library   | shadcn/ui + Tailwind CSS| latest    | Required by requirements; composable, unstyled primitives  |
| Charts       | Recharts                | 2.x       | Required by requirements                                   |
| Data fetching| TanStack Query          | 5.x       | Server-state caching, refetch, optimistic updates          |
| Backend      | Go                      | 1.25      | Required by constraints                                    |
| HTTP router  | chi                     | 5.x       | Lightweight, idiomatic Go; middleware-friendly             |
| Database     | PostgreSQL               | 16        | Required by constraints                                    |
| DB driver    | pgx                     | 5.x       | High-performance native Go PostgreSQL driver               |
| Migrations   | Plain SQL (numbered)    | —         | No magic; fully reviewable in PRs                          |
| Auth         | JWT (HS256)             | —         | Stateless; single-service — symmetric key sufficient       |
| Passwords    | bcrypt                  | —         | Standard; cost factor 12                                   |
| Containers   | Docker + Compose        | latest    | Required by constraints; cloud-agnostic                    |

---

## 5. Data Model

All tables use `uuid` primary keys (`gen_random_uuid()`), `timestamptz` for all timestamps, and `now()` as the default for `created_at` / `updated_at`. `updated_at` is maintained by an `ON UPDATE` trigger on each table.

### ERD (ASCII)

```
users ──< user_roles >── roles ──< role_permissions
  │
  └──< refresh_tokens
  │
  └──< audit_logs (actor_id)
```

---

### 5.1 `users`

| Column          | Type           | Constraints                        | Description                        |
|-----------------|----------------|------------------------------------|------------------------------------|
| `id`            | `uuid`         | PK, default `gen_random_uuid()`    | Surrogate key                      |
| `email`         | `varchar(255)` | UNIQUE, NOT NULL                   | Login identifier                   |
| `name`          | `varchar(255)` | NOT NULL                           | Display name                       |
| `password_hash` | `varchar(255)` | NOT NULL                           | bcrypt hash (cost 12)              |
| `is_active`     | `boolean`      | NOT NULL, default `true`           | `false` blocks login               |
| `created_at`    | `timestamptz`  | NOT NULL, default `now()`          |                                    |
| `updated_at`    | `timestamptz`  | NOT NULL, default `now()`          | Updated by trigger                 |

Indexes: unique on `email`.

> Deactivation uses `is_active = false` (not `deleted_at`) so that foreign keys from `audit_logs` remain intact and data is preserved.

---

### 5.2 `roles`

| Column        | Type           | Constraints                     | Description                            |
|---------------|----------------|---------------------------------|----------------------------------------|
| `id`          | `uuid`         | PK                              |                                        |
| `name`        | `varchar(100)` | UNIQUE, NOT NULL                | e.g. `"admin"`, `"viewer"`             |
| `description` | `text`         | NULLABLE                        | Human-readable explanation             |
| `is_system`   | `boolean`      | NOT NULL, default `false`       | `true` → cannot be deleted             |
| `created_at`  | `timestamptz`  | NOT NULL                        |                                        |
| `updated_at`  | `timestamptz`  | NOT NULL                        |                                        |

Seeded: one system role `admin` with `is_system = true` and all permissions.

---

### 5.3 `user_roles`

| Column        | Type          | Constraints                    | Description              |
|---------------|---------------|--------------------------------|--------------------------|
| `user_id`     | `uuid`        | FK → `users.id`, NOT NULL      |                          |
| `role_id`     | `uuid`        | FK → `roles.id`, NOT NULL      |                          |
| `assigned_at` | `timestamptz` | NOT NULL, default `now()`      |                          |

PK: composite `(user_id, role_id)`. A user may hold multiple roles; effective permissions are the union of all assigned roles.

---

### 5.4 `role_permissions`

| Column       | Type           | Constraints                            | Description                                  |
|--------------|----------------|----------------------------------------|----------------------------------------------|
| `id`         | `uuid`         | PK                                     |                                              |
| `role_id`    | `uuid`         | FK → `roles.id`, NOT NULL              |                                              |
| `resource`   | `varchar(100)` | NOT NULL                               | Feature key: `users`, `roles`, `audit_logs`  |
| `action`     | `varchar(50)`  | NOT NULL                               | `read` \| `write` \| `delete`                |
| `created_at` | `timestamptz`  | NOT NULL                               |                                              |

Unique constraint: `(role_id, resource, action)`. Index on `role_id`.

---

### 5.5 `refresh_tokens`

| Column        | Type           | Constraints                     | Description                              |
|---------------|----------------|---------------------------------|------------------------------------------|
| `id`          | `uuid`         | PK                              |                                          |
| `user_id`     | `uuid`         | FK → `users.id`, NOT NULL       |                                          |
| `token_hash`  | `varchar(255)` | UNIQUE, NOT NULL                | SHA-256 of the raw opaque token          |
| `expires_at`  | `timestamptz`  | NOT NULL                        | `now() + JWT_REFRESH_TTL`                |
| `revoked_at`  | `timestamptz`  | NULLABLE                        | Set on logout or token rotation          |
| `created_at`  | `timestamptz`  | NOT NULL                        |                                          |

Index on `token_hash`. The raw token lives only in the httpOnly cookie; the DB stores only its hash.

---

### 5.6 `audit_logs`

| Column          | Type           | Constraints                     | Description                              |
|-----------------|----------------|---------------------------------|------------------------------------------|
| `id`            | `uuid`         | PK                              |                                          |
| `actor_id`      | `uuid`         | FK → `users.id`, NOT NULL       | User who performed the action            |
| `action`        | `varchar(100)` | NOT NULL                        | e.g. `user.create`, `role.update`        |
| `resource_type` | `varchar(100)` | NOT NULL                        | e.g. `user`, `role`                      |
| `resource_id`   | `uuid`         | NULLABLE                        | Affected record ID                       |
| `before_state`  | `jsonb`        | NULLABLE                        | Snapshot before mutation                 |
| `after_state`   | `jsonb`        | NULLABLE                        | Snapshot after mutation                  |
| `ip_address`    | `inet`         | NULLABLE                        | Client IP                                |
| `created_at`    | `timestamptz`  | NOT NULL, default `now()`       | Immutable; no `updated_at`               |

**No UPDATE or DELETE allowed** on this table at the application layer — enforced by convention and documented in the README. Indexes: `(actor_id, created_at)`, `(resource_type, resource_id)`.

---

## 6. Authentication & Authorization

### 6.1 Auth Flow

**Login**
1. Client sends `POST /api/v1/auth/login` with `{ "email": "...", "password": "..." }`.
2. Server looks up user by email; returns generic `401` if not found.
3. Server compares password against `password_hash` (bcrypt, cost 12); returns generic `401` on mismatch.
4. If `is_active = false`, returns `403 ACCOUNT_DEACTIVATED`.
5. Server generates a signed HS256 access JWT (TTL: `JWT_ACCESS_TTL`, default 15 min).
6. Server generates a cryptographically random opaque refresh token, stores its SHA-256 hash in `refresh_tokens` with `expires_at = now() + JWT_REFRESH_TTL`.
7. Both tokens are set as `HttpOnly; SameSite=Strict; Secure` cookies (`access_token`, `refresh_token`).
8. Response body: `{ "user": { "id", "email", "name", "roles" } }` — no tokens in body.

**Silent Refresh**
1. Any API call returns `401` when the access token is expired.
2. Frontend `api.ts` interceptor automatically calls `POST /api/v1/auth/refresh`.
3. Server reads `refresh_token` cookie, hashes it, looks up the record.
4. If not found, expired, or revoked → returns `401`; frontend redirects to `/login`.
5. On success: marks old token `revoked_at = now()`, issues new access JWT + new refresh token (rotation), sets new cookies.
6. Frontend retries the original request once.

**Logout**
1. Client sends `POST /api/v1/auth/logout`.
2. Server sets `revoked_at = now()` on the refresh token record.
3. Server clears both cookies (Max-Age=0).
4. Returns `204 No Content`.

**Expired access token (direct)**
Any request with a missing or expired `access_token` cookie and no valid `refresh_token` → redirect to `/login` from the frontend.

---

### 6.2 JWT Payload

```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "name": "Jane Smith",
  "iat": 1700000000,
  "exp": 1700000900
}
```

> Roles and permissions are **not** in the JWT. They are fetched from the database on each authenticated request (keyed by `sub`) and enforced in the RBAC middleware. This ensures permission changes take effect immediately — no waiting for token expiry.

---

### 6.3 RBAC Model

Permissions are stored as `(role_id, resource, action)` triples. A user's effective permissions are the **union** of all permissions across all their assigned roles.

**Valid resources and actions:**

| Resource     | Actions                   |
|--------------|---------------------------|
| `users`      | `read`, `write`, `delete` |
| `roles`      | `read`, `write`, `delete` |
| `audit_logs` | `read`                    |

**Permission check flow:**

```
Incoming request
  → JWT middleware
      • Read access_token cookie
      • Verify HS256 signature + expiry
      • Attach user_id to request context
      • 401 on failure
  → RBAC middleware (per-route, declares required resource + action)
      • Load user's roles + permissions from DB (cache by user_id for request lifetime)
      • Compute union of all role permissions
      • Check: does union contain (resource, action)?
      • 403 FORBIDDEN if not
  → Route handler executes
```

**Seeded data (migration):**
- Role: `admin` (`is_system = true`) with all resource/action combinations.
- User: `admin@example.com` / `changeme` — must be changed on first login (documented in README).

---

## 7. API Specification

### Base URL
`/api/v1`

### Error Response Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Human-readable description",
    "details": { "field": "email" }
  }
}
```

**Error codes used throughout:** `VALIDATION_ERROR`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `ACCOUNT_DEACTIVATED`, `INTERNAL_ERROR`.

---

### 7.1 Health

#### `GET /health`
- **Description**: Liveness check. Returns service status and DB connectivity.
- **Auth**: None
- **Response `200`**:
  ```json
  { "status": "ok", "db": "ok" }
  ```

---

### 7.2 Auth

#### `POST /api/v1/auth/login`
- **Description**: Authenticate with email + password. Sets httpOnly cookies.
- **Auth**: None
- **Request**:
  ```json
  { "email": "user@example.com", "password": "secret" }
  ```
- **Response `200`**:
  ```json
  {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "name": "Jane Smith",
      "roles": ["admin"]
    }
  }
  ```
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 400 | `VALIDATION_ERROR` | Missing/malformed fields |
  | 401 | `UNAUTHORIZED` | Invalid email or password |
  | 403 | `ACCOUNT_DEACTIVATED` | Account is inactive |

---

#### `POST /api/v1/auth/refresh`
- **Description**: Rotate refresh token and issue a new access token.
- **Auth**: `refresh_token` cookie
- **Request**: Empty body
- **Response `200`**: Same shape as login response
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 401 | `UNAUTHORIZED` | Missing, expired, or revoked refresh token |

---

#### `POST /api/v1/auth/logout`
- **Description**: Revoke refresh token and clear both cookies.
- **Auth**: Required (access token)
- **Request**: Empty body
- **Response `204`**: No content
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 401 | `UNAUTHORIZED` | Not authenticated |

---

#### `GET /api/v1/auth/me`
- **Description**: Return the authenticated user's profile and roles.
- **Auth**: Required
- **Response `200`**:
  ```json
  {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "name": "Jane Smith",
      "is_active": true,
      "roles": [{ "id": "uuid", "name": "admin" }],
      "created_at": "2024-01-01T00:00:00Z"
    }
  }
  ```
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 401 | `UNAUTHORIZED` | Not authenticated |

---

### 7.3 Users

All endpoints require authentication. Pagination query params: `page` (default 1), `per_page` (default 25, max 100), `search` (name/email substring), `sort_by` (default `created_at`), `sort_dir` (`asc`/`desc`).

#### `GET /api/v1/users`
- **Auth**: `users:read`
- **Response `200`**:
  ```json
  {
    "data": [{ "id": "uuid", "email": "...", "name": "...", "is_active": true, "roles": [...], "created_at": "..." }],
    "meta": { "page": 1, "per_page": 25, "total": 42 }
  }
  ```

---

#### `POST /api/v1/users`
- **Auth**: `users:write`
- **Request**:
  ```json
  { "email": "new@example.com", "name": "Alice", "password": "temp123", "role_id": "uuid" }
  ```
- **Response `201`**: Created user object (same shape as list item)
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 400 | `VALIDATION_ERROR` | Missing/invalid fields |
  | 409 | `CONFLICT` | Email already in use |

---

#### `GET /api/v1/users/:id`
- **Auth**: `users:read`
- **Response `200`**: Full user object
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 404 | `NOT_FOUND` | User does not exist |

---

#### `PUT /api/v1/users/:id`
- **Auth**: `users:write`
- **Request**:
  ```json
  { "name": "Alice Updated", "email": "newemail@example.com", "role_id": "uuid" }
  ```
- **Response `200`**: Updated user object
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 400 | `VALIDATION_ERROR` | Invalid fields |
  | 404 | `NOT_FOUND` | User does not exist |
  | 409 | `CONFLICT` | Email already in use |

---

#### `POST /api/v1/users/:id/deactivate`
- **Auth**: `users:delete`
- **Request**: Empty body
- **Response `200`**: Updated user object with `is_active: false`
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 404 | `NOT_FOUND` | User does not exist |
  | 409 | `CONFLICT` | Cannot deactivate the last active admin |

---

#### `POST /api/v1/users/:id/reactivate`
- **Auth**: `users:write`
- **Request**: Empty body
- **Response `200`**: Updated user object with `is_active: true`
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 404 | `NOT_FOUND` | User does not exist |

---

#### `PUT /api/v1/users/:id/password`
- **Description**: Admin-initiated password reset (no email required).
- **Auth**: `users:write`
- **Request**:
  ```json
  { "password": "newpassword123" }
  ```
- **Response `204`**: No content
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 400 | `VALIDATION_ERROR` | Password too short (min 8 chars) |
  | 404 | `NOT_FOUND` | User does not exist |

---

### 7.4 Roles

#### `GET /api/v1/roles`
- **Auth**: `roles:read`
- **Response `200`**:
  ```json
  {
    "data": [{
      "id": "uuid", "name": "admin", "description": "...", "is_system": true,
      "permissions": [{ "resource": "users", "action": "read" }],
      "created_at": "..."
    }]
  }
  ```

---

#### `POST /api/v1/roles`
- **Auth**: `roles:write`
- **Request**:
  ```json
  {
    "name": "viewer",
    "description": "Read-only access",
    "permissions": [
      { "resource": "users", "action": "read" },
      { "resource": "audit_logs", "action": "read" }
    ]
  }
  ```
- **Response `201`**: Created role object
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 400 | `VALIDATION_ERROR` | Missing name or invalid permission values |
  | 409 | `CONFLICT` | Role name already in use |

---

#### `GET /api/v1/roles/:id`
- **Auth**: `roles:read`
- **Response `200`**: Full role object including permissions
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 404 | `NOT_FOUND` | Role does not exist |

---

#### `PUT /api/v1/roles/:id`
- **Auth**: `roles:write`
- **Description**: Full replacement of permissions list. Changes apply immediately to all users holding this role.
- **Request**:
  ```json
  {
    "name": "viewer",
    "description": "Updated description",
    "permissions": [{ "resource": "users", "action": "read" }]
  }
  ```
- **Response `200`**: Updated role object
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 400 | `VALIDATION_ERROR` | Invalid fields |
  | 404 | `NOT_FOUND` | Role does not exist |
  | 409 | `CONFLICT` | Name already in use by another role |

---

#### `DELETE /api/v1/roles/:id`
- **Auth**: `roles:delete`
- **Response `204`**: No content
- **Error responses**:
  | Status | Code | Condition |
  |--------|------|-----------|
  | 404 | `NOT_FOUND` | Role does not exist |
  | 409 | `CONFLICT` | Role is a system role OR assigned to one or more users |

---

### 7.5 Audit Logs

#### `GET /api/v1/audit-logs`
- **Auth**: `audit_logs:read`
- **Query params**: `page`, `per_page`, `actor_id` (uuid), `resource_type` (string), `action` (string), `from` (ISO8601), `to` (ISO8601), `sort_dir` (`asc`/`desc`, default `desc`)
- **Response `200`**:
  ```json
  {
    "data": [{
      "id": "uuid",
      "actor": { "id": "uuid", "name": "Jane", "email": "jane@example.com" },
      "action": "user.create",
      "resource_type": "user",
      "resource_id": "uuid",
      "before_state": null,
      "after_state": { "email": "new@example.com", "name": "Bob" },
      "ip_address": "192.168.1.1",
      "created_at": "2024-01-01T12:00:00Z"
    }],
    "meta": { "page": 1, "per_page": 25, "total": 300 }
  }
  ```

---

#### `GET /api/v1/audit-logs/export`
- **Auth**: `audit_logs:read`
- **Description**: Streams a CSV file of filtered audit log entries. Accepts same filter query params as the list endpoint (no pagination — exports all matching rows).
- **Response `200`**:
  - `Content-Type: text/csv`
  - `Content-Disposition: attachment; filename="audit-log-{timestamp}.csv"`
  - CSV columns: `id, actor_name, actor_email, action, resource_type, resource_id, ip_address, created_at`

---

## 8. Frontend Architecture

### 8.1 Routing Structure

| Path               | Component         | Auth Required | Permission Required |
|--------------------|-------------------|---------------|---------------------|
| `/login`           | `LoginPage`       | No            | —                   |
| `/`                | Redirect          | Yes           | Any authenticated   |
| `/dashboard`       | `DashboardPage`   | Yes           | Any authenticated   |
| `/users`           | `UsersListPage`   | Yes           | `users:read`        |
| `/users/new`       | `UserCreatePage`  | Yes           | `users:write`       |
| `/users/:id`       | `UserEditPage`    | Yes           | `users:read`        |
| `/roles`           | `RolesListPage`   | Yes           | `roles:read`        |
| `/roles/new`       | `RoleCreatePage`  | Yes           | `roles:write`       |
| `/roles/:id`       | `RoleEditPage`    | Yes           | `roles:read`        |
| `/audit-logs`      | `AuditLogsPage`   | Yes           | `audit_logs:read`   |
| `*`                | `NotFoundPage`    | No            | —                   |

**`<ProtectedRoute>`**: Reads auth state from `AuthContext`. If not authenticated → redirect to `/login`. If authenticated but missing required permission → render a `403` message component in place of the page.

---

### 8.2 State Management

| State category | Where it lives | Notes |
|----------------|----------------|-------|
| Current user + permissions | `AuthContext` (React Context) | Populated from `GET /auth/me` on app load; cleared on logout |
| Server data (users, roles, audit logs) | TanStack Query cache | Each feature module owns its query/mutation hooks in `useUsers.ts`, `useRoles.ts`, etc. |
| Form state | Local `useState` / React Hook Form | Not lifted unless required |
| Theme preference | `localStorage` (`theme: "light" \| "dark"`) | Read on app init; toggled via `TopBar` |

---

### 8.3 API Client

`src/lib/api.ts` exports a base `request()` function and typed resource helpers.

**Behaviour:**
- Base URL read from `import.meta.env.VITE_API_URL`.
- All requests include `credentials: "include"` (sends cookies automatically).
- Requests default to `Content-Type: application/json`.
- On `401` response: automatically calls `POST /api/v1/auth/refresh` **once**. If refresh succeeds, retries the original request. If refresh also returns `401`, clears `AuthContext` and redirects to `/login`.
- Non-2xx responses are parsed into the standard error envelope and thrown as typed `ApiError` objects for TanStack Query to surface.

---

### 8.4 Component Structure

**`<App>`**
```
<QueryClientProvider>
  <AuthProvider>        ← AuthContext; fetches /auth/me on mount
    <ThemeProvider>     ← Reads localStorage, applies Tailwind dark class
      <RouterProvider>  ← React Router
        <ProtectedRoute>
          <AppLayout>   ← Sidebar + TopBar + <Outlet>
            <Page />
```

**Shared components available to all feature pages:**
- `<DataTable columns={...} data={...} pagination={...} />` — generic, fully typed
- `<LineChart />`, `<BarChart />`, `<PieChart />` — thin Recharts wrappers with consistent styling
- All shadcn/ui primitives: `Button`, `Input`, `Select`, `Dialog`, `Badge`, `Card`, `Table`, `Tooltip`, etc.

**Per-project theming:** Override CSS variables in `src/theme.css`. The file is imported in `main.tsx` after shadcn's base styles, so any variable defined there wins. Switching between light and dark applies the `dark` class to `<html>` — no page reload required.

---

## 9. Environment Variables

### Backend

| Variable           | Required | Default  | Description                                      |
|--------------------|----------|----------|--------------------------------------------------|
| `DATABASE_URL`     | Yes      | —        | PostgreSQL connection string (DSN format)        |
| `JWT_SECRET`       | Yes      | —        | HS256 signing secret, minimum 32 characters      |
| `JWT_ACCESS_TTL`   | No       | `15m`    | Access token TTL (Go duration string)            |
| `JWT_REFRESH_TTL`  | No       | `720h`   | Refresh token TTL — 30 days (Go duration string) |
| `CORS_ORIGIN`      | Yes      | —        | Allowed frontend origin, e.g. `http://localhost:5173` |
| `PORT`             | No       | `8080`   | HTTP server listen port                          |

### Frontend

| Variable         | Required | Default                  | Description                    |
|------------------|----------|--------------------------|--------------------------------|
| `VITE_API_URL`   | Yes      | `http://localhost:8080`  | Backend base URL               |

---

## 10. Local Development Setup

### Prerequisites
- Docker + Docker Compose v2
- Make

### Steps

```bash
# 1. Clone the repo
git clone https://github.com/your-org/saas-foundation.git
cd saas-foundation

# 2. Copy env example and fill in secrets
cp .env.example .env
# Edit .env: set JWT_SECRET to any random 32+ char string

# 3. Start the full stack with hot reload
make dev
# Starts: postgres:5432, backend:8080 (air), frontend:5173 (Vite HMR)

# 4. Run database migrations
make migrate

# 5. Seed the default admin role and user
make seed
# Creates: admin@example.com / changeme — change password immediately

# 6. Open the app
open http://localhost:5173
```

### Makefile Targets

| Target | Description |
|--------|-------------|
| `make dev` | `docker compose -f docker-compose.yml -f docker-compose.dev.yml up` |
| `make build` | Build all Docker images |
| `make migrate` | Run pending SQL migrations against `DATABASE_URL` |
| `make test` | Run Go unit + integration tests |
| `make seed` | Insert default admin role + admin user (idempotent) |

### Docker Compose Services

| Service | Image | Ports | Notes |
|---------|-------|-------|-------|
| `postgres` | `postgres:16-alpine` | 5432 | Named volume `pgdata` |
| `backend` | Built from `backend/Dockerfile` | 8080 | Dev: bind-mount source + air |
| `frontend` | Built from `frontend/Dockerfile` | 5173 (dev) / 80 (prod) | Dev: Vite HMR; prod: nginx static |

---

## 11. Implementation Plan

> Complexity: **S** = < 2 hrs · **M** = 2–8 hrs · **L** = 1–3 days

### Phase 1 — MVP: Auth + User Management + Role Builder

| # | Task | Complexity | Depends On |
|---|------|------------|------------|
| 1.1 | Repo scaffold: monorepo dirs, Makefile, Docker Compose files, `.env.example` | S | — |
| 1.2 | DB migrations: all 6 tables, indexes, `updated_at` triggers, seed admin role + user | S | 1.1 |
| 1.3 | Go bootstrap: HTTP server, config loader, DB connection pool, `GET /health` | S | 1.1 |
| 1.4 | Auth API: `POST /auth/login`, `POST /auth/logout`, `POST /auth/refresh`, `GET /auth/me` | M | 1.3, 1.2 |
| 1.5 | JWT + RBAC middleware: token validation, permission lookup, request context injection | M | 1.4 |
| 1.6 | Users API: all 7 endpoints including deactivate guard (last admin check) | M | 1.5 |
| 1.7 | Roles API: all 5 endpoints including deletion guard (in-use check) | M | 1.5 |
| 1.8 | Frontend scaffold: Vite + React + Tailwind + shadcn/ui setup, React Router, TanStack Query | S | 1.1 |
| 1.9 | Auth layer: `AuthContext`, `LoginPage`, token refresh interceptor in `api.ts`, `ProtectedRoute` | M | 1.8, 1.4 |
| 1.10 | Navigation shell: `AppLayout`, `Sidebar`, `TopBar`, breadcrumb stub | S | 1.9 |
| 1.11 | User Management UI: `UsersListPage`, `UserCreatePage`, `UserEditPage`, deactivate/reactivate actions | M | 1.10, 1.6 |
| 1.12 | Role Builder UI: `RolesListPage`, `RoleCreatePage`, `RoleEditPage` with permission matrix | M | 1.10, 1.7 |

---

### Phase 2 — UI Component Library + Audit Logs + Theming

| # | Task | Complexity | Depends On |
|---|------|------------|------------|
| 2.1 | Audit log API: `GET /audit-logs` (paginated + filterable) + `GET /audit-logs/export` (CSV stream) | M | 1.5 |
| 2.2 | Audit log middleware: auto-write on all state-changing API calls (users + roles mutations) | M | 2.1 |
| 2.3 | Audit Log UI: `AuditLogsPage` with filter controls + CSV export button | M | 2.1, 1.10 |
| 2.4 | `DataTable` component: sortable columns, client-side filter, pagination, configurable column defs | M | 1.8 |
| 2.5 | `Charts` components: `LineChart`, `BarChart`, `PieChart` wrappers with consistent theme-aware styling | S | 1.8 |
| 2.6 | `Forms` components: `FormField`, `FormError`, `FormHelperText` wrappers for controlled inputs | S | 1.8 |
| 2.7 | `Navigation` refinement: `Breadcrumbs` component, permission-aware sidebar link visibility, collapsible sidebar | S | 1.10 |
| 2.8 | Dark mode + theming: CSS variable system in `theme.css`, `localStorage` persistence, `TopBar` toggle | S | 1.10 |
| 2.9 | Developer docs: README (setup, architecture, how to add a feature module), component usage guide | S | 2.4–2.8 |

---

## 12. Open Questions & Deferred Decisions

| # | Question / Decision | Impact | Status |
|---|---------------------|--------|--------|
| 1 | Multi-tenancy: confirmed out of scope for v1. If needed in v2, every table except `audit_logs` requires a `tenant_id` column and row-level security. | Major schema migration | Deferred to v2 |
| 2 | Forgot password flow: deferred. Requires an SMTP/email service integration. In v1, admins reset passwords via `PUT /users/:id/password`. | New API endpoint + email service | Deferred to v2 |
| 3 | Deployment target: Docker Compose only. If a cloud platform is chosen later, a CI/CD pipeline and secrets manager will need to be configured. | Infra + secrets management | Open |
| 4 | Failed/denied action logging: not captured in v1. If compliance requirements emerge, the audit middleware must be extended to log 401/403 responses. | Audit log schema + middleware | Deferred to v2 |
| 5 | JWT algorithm: HS256 with a single shared secret. If the backend is ever split into multiple services that need to independently verify tokens, migrate to RS256. | Key management + token verification | Deferred if needed |
| 6 | Audit log retention: logs are retained indefinitely in v1. A purge policy or archival strategy should be revisited when data volume becomes a concern. | DB storage + potential GDPR impact | Deferred to v2 |
