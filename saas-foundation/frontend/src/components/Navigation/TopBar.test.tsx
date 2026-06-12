import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { TopBar } from "./TopBar";
import { AuthContext, type AuthContextValue } from "@/features/auth/AuthContext";
import type { MeUser } from "@/lib/api";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const mockNavigate = vi.fn();

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return { ...actual, useNavigate: () => mockNavigate };
});

const testUser: MeUser = {
  id: "user-1",
  email: "jane@example.com",
  name: "Jane Smith",
  is_active: true,
  roles: [{ id: "role-1", name: "admin" }],
  created_at: "2024-01-01T00:00:00Z",
};

function makeCtx(overrides: Partial<AuthContextValue> = {}): AuthContextValue {
  return {
    user: testUser,
    isLoading: false,
    hasPermission: vi.fn().mockReturnValue(true),
    login: vi.fn(),
    logout: vi.fn(),
    ...overrides,
  };
}

function renderTopBar(ctx: AuthContextValue) {
  return render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter initialEntries={["/dashboard"]}>
        <TopBar />
      </MemoryRouter>
    </AuthContext.Provider>
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("TopBar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("displays the user's name", () => {
    renderTopBar(makeCtx());
    expect(screen.getByText("Jane Smith")).toBeInTheDocument();
  });

  it("shows user name and email in the dropdown", async () => {
    renderTopBar(makeCtx());
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: /user menu/i }));

    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
  });

  it("renders a Sign out menu item in the dropdown", async () => {
    renderTopBar(makeCtx());
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: /user menu/i }));

    expect(screen.getByRole("menuitem", { name: /sign out/i })).toBeInTheDocument();
  });

  it("calls logout and navigates to /login on sign out", async () => {
    const logout = vi.fn().mockResolvedValue(undefined);
    renderTopBar(makeCtx({ logout }));
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: /user menu/i }));
    await user.click(screen.getByRole("menuitem", { name: /sign out/i }));

    await waitFor(() => {
      expect(logout).toHaveBeenCalledOnce();
      expect(mockNavigate).toHaveBeenCalledWith("/login", { replace: true });
    });
  });
});
