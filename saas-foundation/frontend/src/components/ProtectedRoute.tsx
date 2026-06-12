import { Navigate } from "react-router";
import { useAuth } from "@/features/auth/useAuth";
import { Spinner } from "@/components/ui/spinner";

interface ProtectedRouteProps {
  /**
   * Optional permission required to view this route, in "resource:action" format.
   * e.g. "users:read", "roles:write".
   * Omit to require only authentication (no specific permission).
   */
  permission?: string;
  children: React.ReactNode;
}

/**
 * ProtectedRoute renders children only when the current user is authenticated
 * and has the required permission.
 *
 * - While auth state is loading → renders a centred spinner.
 * - Unauthenticated users → redirected to /login.
 * - Authenticated users lacking the required permission → 403 message.
 * - Otherwise → renders children.
 */
export function ProtectedRoute({ permission, children }: ProtectedRouteProps) {
  const { user, isLoading, hasPermission } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner label="Checking authentication…" />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" replace />;
  }

  if (permission) {
    const [resource, action] = permission.split(":");
    if (!hasPermission(resource ?? "", action ?? "")) {
      return (
        <div className="flex min-h-screen items-center justify-center">
          <div className="text-center">
            <h1 className="text-4xl font-bold">403</h1>
            <p className="mt-2 text-muted-foreground">
              You do not have permission to view this page.
            </p>
          </div>
        </div>
      );
    }
  }

  return <>{children}</>;
}
