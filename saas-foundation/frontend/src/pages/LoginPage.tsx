import { Navigate } from "react-router";
import { LoginForm } from "@/features/auth/LoginForm";
import { useAuth } from "@/features/auth/useAuth";
import { Spinner } from "@/components/ui/spinner";

/**
 * LoginPage renders the login form and handles redirects.
 * Already-authenticated users are sent straight to /dashboard.
 */
export function LoginPage() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner />
      </div>
    );
  }

  if (user) {
    return <Navigate to="/dashboard" replace />;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 px-4">
      <LoginForm />
    </div>
  );
}
