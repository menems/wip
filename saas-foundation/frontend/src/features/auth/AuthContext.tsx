import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { authApi, rolesApi, type MeUser } from "@/lib/api";

// ---------------------------------------------------------------------------
// Context value type
// ---------------------------------------------------------------------------

export interface AuthContextValue {
  /** The currently authenticated user, or null if not signed in. */
  user: MeUser | null;
  /** True while the initial /auth/me request is in-flight on mount. */
  isLoading: boolean;
  /**
   * Checks whether the current user has the given resource+action permission.
   * Returns false if no user is authenticated or permissions have not loaded.
   *
   * NOTE: permissions are computed by fetching /api/v1/roles after /auth/me.
   * If the user lacks `roles:read` the permission set will be empty.
   * A future enhancement is to include effective permissions in /auth/me.
   */
  hasPermission: (resource: string, action: string) => boolean;
  /** Sign in with email + password. Resolves on success, throws ApiError on failure. */
  login: (email: string, password: string) => Promise<void>;
  /** Revoke the refresh token and clear local auth state. */
  logout: () => Promise<void>;
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

export const AuthContext = createContext<AuthContextValue | null>(null);

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

/**
 * AuthProvider fetches the current user from /auth/me on mount and provides
 * the auth state + helpers to the entire component tree.
 *
 * Place it inside QueryClientProvider but outside RouterProvider:
 *   <QueryClientProvider>
 *     <AuthProvider>
 *       <RouterProvider router={router} />
 *     </AuthProvider>
 *   </QueryClientProvider>
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<MeUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  // "resource:action" strings — e.g. "users:read", "roles:write"
  const [permissions, setPermissions] = useState<ReadonlySet<string>>(
    new Set()
  );

  /**
   * Loads the user from /auth/me, then loads permissions from /api/v1/roles.
   * If either request fails the state is reset to unauthenticated.
   */
  const loadUser = useCallback(async () => {
    setIsLoading(true);
    try {
      const { user: me } = await authApi.me();
      setUser(me);
      await loadPermissions(me.roles);
    } catch {
      setUser(null);
      setPermissions(new Set());
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Populate permissions by fetching all roles and computing the union of the
  // authenticated user's role permissions.
  async function loadPermissions(
    userRoles: Array<{ id: string; name: string }>
  ) {
    try {
      const { data: allRoles } = await rolesApi.list();
      const userRoleIds = new Set(userRoles.map((r) => r.id));
      const effective = new Set<string>();
      for (const role of allRoles) {
        if (userRoleIds.has(role.id)) {
          for (const perm of role.permissions) {
            effective.add(`${perm.resource}:${perm.action}`);
          }
        }
      }
      setPermissions(effective);
    } catch {
      // If roles cannot be fetched (e.g. 403) permissions remain empty.
      setPermissions(new Set());
    }
  }

  // Track whether we've already subscribed to avoid double-registering in StrictMode.
  const listenerAdded = useRef(false);

  useEffect(() => {
    void loadUser();

    if (!listenerAdded.current) {
      listenerAdded.current = true;
      const handleUnauthenticated = () => {
        setUser(null);
        setPermissions(new Set());
      };
      window.addEventListener("auth:unauthenticated", handleUnauthenticated);
      return () => {
        window.removeEventListener(
          "auth:unauthenticated",
          handleUnauthenticated
        );
        listenerAdded.current = false;
      };
    }
  }, [loadUser]);

  const login = useCallback(
    async (email: string, password: string) => {
      await authApi.login(email, password);
      // Re-fetch the full user profile + permissions after a successful login.
      await loadUser();
    },
    [loadUser]
  );

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      // Ignore network/API errors — the session is gone from the client's
      // perspective regardless. Always clear local state.
    } finally {
      setUser(null);
      setPermissions(new Set());
    }
  }, []);

  const hasPermission = useCallback(
    (resource: string, action: string) =>
      permissions.has(`${resource}:${action}`),
    [permissions]
  );

  return (
    <AuthContext.Provider
      value={{ user, isLoading, hasPermission, login, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

/**
 * useAuth returns the AuthContext value.
 * Must be called inside a component rendered within AuthProvider.
 */
export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within <AuthProvider>");
  }
  return ctx;
}
