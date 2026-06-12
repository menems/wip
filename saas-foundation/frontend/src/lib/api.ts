/**
 * api.ts — base HTTP client and typed resource helpers.
 *
 * Every request includes `credentials: "include"` so httpOnly cookies are sent
 * automatically. On a 401 the client silently refreshes the access token and
 * retries the original request once. If the refresh also fails it dispatches
 * the `"auth:unauthenticated"` window event so AuthContext can clear state and
 * redirect to /login.
 */

const BASE_URL = (import.meta.env["VITE_API_URL"] as string | undefined) ?? "";

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

/** Shape of the error envelope the backend always returns for non-2xx. */
interface ApiErrorBody {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}

/**
 * ApiError is thrown for all non-2xx HTTP responses.
 * `code` is the backend's machine-readable error code (e.g. "NOT_FOUND").
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly details?: Record<string, unknown>
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// ---------------------------------------------------------------------------
// Core request function
// ---------------------------------------------------------------------------

/**
 * Guards concurrent refresh calls: only one refresh is in-flight at a time.
 * All callers that receive a 401 while a refresh is pending share the same
 * Promise so we do not fire duplicate refresh requests.
 */
let pendingRefresh: Promise<void> | null = null;

/** Issues a token refresh. Throws ApiError on failure. */
async function performRefresh(): Promise<void> {
  const res = await fetch(`${BASE_URL}/api/v1/auth/refresh`, {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) {
    throw new ApiError(res.status, "UNAUTHORIZED", "Session expired");
  }
}

/** Parses an error response body and throws the resulting ApiError. */
async function parseErrorAndThrow(res: Response): Promise<never> {
  let code = "INTERNAL_ERROR";
  let message = `Request failed with status ${res.status}`;
  let details: Record<string, unknown> | undefined;

  try {
    const body = (await res.json()) as ApiErrorBody;
    code = body.error.code;
    message = body.error.message;
    details = body.error.details;
  } catch {
    // JSON parse failed — keep defaults
  }

  throw new ApiError(res.status, code, message, details);
}

/**
 * Core fetch wrapper used by all API helpers.
 *
 * @param path    Absolute path from the API root, e.g. "/api/v1/users"
 * @param options Standard fetch RequestInit; Content-Type defaults to JSON
 * @param isRetry Internal flag — set to true on the post-refresh retry
 */
export async function request<T>(
  path: string,
  options?: RequestInit,
  isRetry = false
): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (res.status === 401 && !isRetry) {
    // Coalesce concurrent 401 handling into a single refresh attempt.
    pendingRefresh ??= performRefresh().finally(() => {
      pendingRefresh = null;
    });

    try {
      await pendingRefresh;
      return request<T>(path, options, true);
    } catch {
      window.dispatchEvent(new CustomEvent("auth:unauthenticated"));
      throw new ApiError(401, "UNAUTHORIZED", "Session expired");
    }
  }

  if (!res.ok) {
    await parseErrorAndThrow(res);
  }

  // 204 No Content — return undefined cast to T (callers type as void/undefined)
  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

// ---------------------------------------------------------------------------
// Auth types & endpoints
// ---------------------------------------------------------------------------

/** User shape returned by /auth/login. */
export interface LoginUser {
  id: string;
  email: string;
  name: string;
  roles: string[]; // role names (strings) per spec
}

/** /auth/login response */
export interface LoginResponse {
  user: LoginUser;
}

/** Full user shape returned by /auth/me. */
export interface MeUser {
  id: string;
  email: string;
  name: string;
  is_active: boolean;
  roles: Array<{ id: string; name: string }>;
  created_at: string;
}

/** /auth/me response */
export interface MeResponse {
  user: MeUser;
}

/** Auth API resource helpers. */
export const authApi = {
  /** Authenticate with email + password. Sets httpOnly cookies on success. */
  login: (email: string, password: string) =>
    request<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),

  /** Revoke the refresh token and clear both cookies. Returns void (204). */
  logout: () => request<void>("/api/v1/auth/logout", { method: "POST" }),

  /** Return the authenticated user's profile and roles. */
  me: () => request<MeResponse>("/api/v1/auth/me"),
};

// ---------------------------------------------------------------------------
// Role types & endpoints (used by AuthContext to compute permissions)
// ---------------------------------------------------------------------------

/** A permission triple attached to a role. */
export interface RolePermission {
  resource: string;
  action: string;
}

/** Full role object including permissions. */
export interface RoleItem {
  id: string;
  name: string;
  description: string | null;
  is_system: boolean;
  permissions: RolePermission[];
  created_at: string;
}

/** /api/v1/roles list response */
export interface RolesListResponse {
  data: RoleItem[];
}

export interface CreateRoleBody {
  name: string;
  description?: string;
  permissions: Array<{ resource: string; action: string }>;
}

export interface UpdateRoleBody {
  name: string;
  description?: string;
  permissions: Array<{ resource: string; action: string }>;
}

/** Roles API resource helpers. */
export const rolesApi = {
  /** List all roles including their permissions. Requires roles:read. */
  list: () => request<RolesListResponse>("/api/v1/roles"),

  /** Fetch a single role by ID including its permissions. Requires roles:read. */
  get: (id: string) => request<RoleItem>(`/api/v1/roles/${id}`),

  /** Create a new role with a permission set. Requires roles:write. */
  create: (body: CreateRoleBody) =>
    request<RoleItem>("/api/v1/roles", {
      method: "POST",
      body: JSON.stringify(body),
    }),

  /** Replace a role's name, description, and full permission set. Requires roles:write. */
  update: (id: string, body: UpdateRoleBody) =>
    request<RoleItem>(`/api/v1/roles/${id}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),

  /** Delete a role. Blocked for system roles and roles assigned to users. Requires roles:delete. */
  delete: (id: string) =>
    request<void>(`/api/v1/roles/${id}`, { method: "DELETE" }),
};

// ---------------------------------------------------------------------------
// Users & audit-log endpoint stubs (implemented in tasks 1.11 and 2.1)
// ---------------------------------------------------------------------------

export interface UserRole {
  id: string;
  name: string;
}

export interface UserItem {
  id: string;
  email: string;
  name: string;
  is_active: boolean;
  roles: UserRole[];
  created_at: string;
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
}

export interface UsersListResponse {
  data: UserItem[];
  meta: PaginationMeta;
}

export interface UserFilter {
  page?: number;
  per_page?: number;
  search?: string;
  sort_by?: string;
  sort_dir?: "asc" | "desc";
}

export interface CreateUserBody {
  email: string;
  name: string;
  password: string;
  role_id: string;
}

export interface UpdateUserBody {
  name?: string;
  email?: string;
  role_id?: string;
}

/** Users API resource helpers. */
export const usersApi = {
  list: (filter: UserFilter = {}) => {
    const params = new URLSearchParams();
    if (filter.page) params.set("page", String(filter.page));
    if (filter.per_page) params.set("per_page", String(filter.per_page));
    if (filter.search) params.set("search", filter.search);
    if (filter.sort_by) params.set("sort_by", filter.sort_by);
    if (filter.sort_dir) params.set("sort_dir", filter.sort_dir);
    const qs = params.toString();
    return request<UsersListResponse>(`/api/v1/users${qs ? `?${qs}` : ""}`);
  },
  // Backend returns the user object directly (not wrapped in { user: ... })
  get: (id: string) => request<UserItem>(`/api/v1/users/${id}`),
  create: (body: CreateUserBody) =>
    request<UserItem>("/api/v1/users", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  update: (id: string, body: UpdateUserBody) =>
    request<UserItem>(`/api/v1/users/${id}`, {
      method: "PUT",
      body: JSON.stringify(body),
    }),
  deactivate: (id: string) =>
    request<UserItem>(`/api/v1/users/${id}/deactivate`, { method: "POST" }),
  reactivate: (id: string) =>
    request<UserItem>(`/api/v1/users/${id}/reactivate`, { method: "POST" }),
  resetPassword: (id: string, password: string) =>
    request<void>(`/api/v1/users/${id}/password`, {
      method: "PUT",
      body: JSON.stringify({ password }),
    }),
};
