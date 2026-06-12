import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { ProtectedRoute } from "./ProtectedRoute";
import { AuthContext, type AuthContextValue } from "@/features/auth/AuthContext";
import type { MeUser } from "@/lib/api";

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const adminUser: MeUser = {
  id: "user-1",
  email: "admin@example.com",
  name: "Admin",
  is_active: true,
  roles: [{ id: "role-1", name: "admin" }],
  created_at: "2024-01-01T00:00:00Z",
};

function makeAuthContext(overrides: Partial<AuthContextValue>): AuthContextValue {
  return {
    user: null,
    isLoading: false,
    hasPermission: vi.fn().mockReturnValue(false),
    login: vi.fn(),
    logout: vi.fn(),
    ...overrides,
  };
}

/** Renders ProtectedRoute inside a MemoryRouter so Navigate works. */
function renderProtectedRoute(
  ctx: AuthContextValue,
  permission?: string,
  initialPath = "/protected"
) {
  return render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route
            path="/protected"
            element={
              <ProtectedRoute permission={permission}>
                <div>Protected content</div>
              </ProtectedRoute>
            }
          />
          <Route path="/login" element={<div>Login page</div>} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("ProtectedRoute", () => {
  it("renders a loading spinner while auth state is loading", () => {
    const ctx = makeAuthContext({ isLoading: true });
    renderProtectedRoute(ctx);

    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.queryByText("Protected content")).not.toBeInTheDocument();
  });

  it("redirects to /login when user is not authenticated", () => {
    const ctx = makeAuthContext({ user: null, isLoading: false });
    renderProtectedRoute(ctx);

    expect(screen.getByText("Login page")).toBeInTheDocument();
    expect(screen.queryByText("Protected content")).not.toBeInTheDocument();
  });

  it("renders children when authenticated and no permission is required", () => {
    const ctx = makeAuthContext({ user: adminUser });
    renderProtectedRoute(ctx);

    expect(screen.getByText("Protected content")).toBeInTheDocument();
  });

  it("renders 403 when user lacks the required permission", () => {
    const ctx = makeAuthContext({
      user: adminUser,
      hasPermission: vi.fn().mockReturnValue(false),
    });
    renderProtectedRoute(ctx, "users:write");

    expect(screen.getByText("403")).toBeInTheDocument();
    expect(screen.queryByText("Protected content")).not.toBeInTheDocument();
  });

  it("renders children when user has the required permission", () => {
    const ctx = makeAuthContext({
      user: adminUser,
      hasPermission: vi.fn().mockReturnValue(true),
    });
    renderProtectedRoute(ctx, "users:read");

    expect(screen.getByText("Protected content")).toBeInTheDocument();
    expect(screen.queryByText("403")).not.toBeInTheDocument();
  });

  it("calls hasPermission with the correct resource and action", () => {
    const hasPermission = vi.fn().mockReturnValue(true);
    const ctx = makeAuthContext({ user: adminUser, hasPermission });
    renderProtectedRoute(ctx, "audit_logs:read");

    expect(hasPermission).toHaveBeenCalledWith("audit_logs", "read");
  });
});
