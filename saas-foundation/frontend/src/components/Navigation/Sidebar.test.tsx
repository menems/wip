import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { Sidebar } from "./Sidebar";
import { AuthContext, type AuthContextValue } from "@/features/auth/AuthContext";
import type { MeUser } from "@/lib/api";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const adminUser: MeUser = {
  id: "user-1",
  email: "admin@example.com",
  name: "Admin",
  is_active: true,
  roles: [{ id: "role-1", name: "admin" }],
  created_at: "2024-01-01T00:00:00Z",
};

function makeCtx(hasPermissionFn: (r: string, a: string) => boolean): AuthContextValue {
  return {
    user: adminUser,
    isLoading: false,
    hasPermission: vi.fn().mockImplementation(hasPermissionFn),
    login: vi.fn(),
    logout: vi.fn(),
  };
}

function renderSidebar(ctx: AuthContextValue) {
  return render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter initialEntries={["/dashboard"]}>
        <Sidebar />
      </MemoryRouter>
    </AuthContext.Provider>
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("Sidebar", () => {
  it("always renders the Dashboard link (no permission required)", () => {
    const ctx = makeCtx(() => false); // no permissions
    renderSidebar(ctx);
    expect(screen.getByRole("link", { name: /dashboard/i })).toBeInTheDocument();
  });

  it("renders permission-gated links when the user has permission", () => {
    const ctx = makeCtx(() => true); // all permissions
    renderSidebar(ctx);

    expect(screen.getByRole("link", { name: /users/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /roles/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /audit logs/i })).toBeInTheDocument();
  });

  it("hides links the user does not have permission for", () => {
    // Only grant users:read, not roles:read or audit_logs:read
    const ctx = makeCtx((resource) => resource === "users");
    renderSidebar(ctx);

    expect(screen.getByRole("link", { name: /users/i })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /roles/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /audit logs/i })).not.toBeInTheDocument();
  });

  it("calls hasPermission with correct resource and action for each nav item", () => {
    const ctx = makeCtx(() => true);
    renderSidebar(ctx);

    const hasPermission = vi.mocked(ctx.hasPermission);
    expect(hasPermission).toHaveBeenCalledWith("users", "read");
    expect(hasPermission).toHaveBeenCalledWith("roles", "read");
    expect(hasPermission).toHaveBeenCalledWith("audit_logs", "read");
  });

  it("hides text labels when collapsed, shows them when expanded", async () => {
    const ctx = makeCtx(() => true);
    renderSidebar(ctx);
    const user = userEvent.setup();

    // Initially expanded — labels visible
    expect(screen.getByText("Dashboard")).toBeVisible();

    // Click collapse button
    await user.click(
      screen.getByRole("button", { name: /collapse sidebar/i })
    );

    // Labels hidden (rendered but not visible due to CSS, or not rendered)
    expect(screen.queryByText("Dashboard")).not.toBeInTheDocument();
  });

  it("toggles the aria-label on the collapse button", async () => {
    const ctx = makeCtx(() => true);
    renderSidebar(ctx);
    const user = userEvent.setup();

    const btn = screen.getByRole("button", { name: /collapse sidebar/i });
    await user.click(btn);
    expect(
      screen.getByRole("button", { name: /expand sidebar/i })
    ).toBeInTheDocument();
  });
});
