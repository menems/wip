# Frontend Architecture

## Feature module layout

```
features/<domain>/
  use<Domain>.ts    ← TanStack Query hooks. Query keys exported as const.
  <Domain>Form.tsx  ← Controlled form (useState); client validation first.
  <Domain>Table.tsx ← Pure presentational; data + callbacks via props.
pages/
  <Domain>ListPage.tsx
  <Domain>CreatePage.tsx
  <Domain>EditPage.tsx
```

## API client (`src/lib/api.ts`)

One `request<T>()` function with 401→refresh→retry built in.
Resource helpers (`usersApi`, `rolesApi`, `authApi`) are thin wrappers.
Non-2xx throws `ApiError` (`.status`, `.code`, `.message`).

## Auth context (`src/features/auth/AuthContext.tsx`)

`AuthProvider` fetches `/auth/me` on mount, then `/api/v1/roles` to build a `Set<"resource:action">`.
`hasPermission(resource, action)` checks membership.
Consumed via `useAuth()`.

## Routing (`src/router.tsx`)

All authenticated routes are children of `ProtectedRoute > AppLayout`.
Per-page permission gates: second `<ProtectedRoute permission="...">` wrapper.

## Testing

Vitest + React Testing Library.
Mock at `@/lib/api` boundary (`vi.mock("@/lib/api", ...)`) for hook tests.
Mock hooks (`vi.mock("@/features/.../useX")`) for page tests.
Hook tests need `QueryClientProvider` wrapper with `retry: false`.
