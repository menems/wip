import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { RolesListPage } from "./RolesListPage";
import { AuthContext, type AuthContextValue } from "@/features/auth/AuthContext";
import { ApiError, type MeUser, type RoleItem } from "@/lib/api";

// ---------------------------------------------------------------------------
// Mock role hooks
// ---------------------------------------------------------------------------

vi.mock("@/features/roles/useRoles", () => ({
  useRoles: vi.fn(),
  useDeleteRole: vi.fn(),
}));

import { useRoles, useDeleteRole } from "@/features/roles/useRoles";

const mockUseRoles = vi.mocked(useRoles);
const mockUseDeleteRole = vi.mocked(useDeleteRole);

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const adminUser: MeUser = {
  id: "user-1",
  email: "admin@example.com",
  name: "Admin",
  is_active: true,
  roles: [{ id: "role-1", name: "admin" }],
  created_at: "2024-01-01T00:00:00Z",
};

const roles: RoleItem[] = [
  {
    id: "role-1",
    name: "admin",
    description: "Administrator",
    is_system: true,
    permissions: [
      { resource: "users", action: "read" },
      { resource: "users", action: "write" },
    ],
    created_at: "2024-01-01T00:00:00Z",
  },
  {
    id: "role-2",
    name: "viewer",
    description: "Read-only",
    is_system: false,
    permissions: [{ resource: "users", action: "read" }],
    created_at: "2024-01-02T00:00:00Z",
  },
];

const stubMutation = {
  mutate: vi.fn(),
  mutateAsync: vi.fn().mockResolvedValue(undefined),
  isPending: false,
  isSuccess: false,
  isError: false,
  isIdle: true,
  error: null,
  data: undefined,
  reset: vi.fn(),
  context: undefined,
  variables: undefined,
  failureCount: 0,
  failureReason: null,
  status: "idle" as const,
  submittedAt: 0,
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeAuthCtx(
  hasPermissionFn: (resource: string, action: string) => boolean
): AuthContextValue {
  return {
    user: adminUser,
    isLoading: false,
    hasPermission: vi.fn().mockImplementation(hasPermissionFn),
    login: vi.fn(),
    logout: vi.fn(),
  };
}

function renderPage(ctx: AuthContextValue = makeAuthCtx(() => true)) {
  return render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter>
        <RolesListPage />
      </MemoryRouter>
    </AuthContext.Provider>
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("RolesListPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockUseDeleteRole.mockReturnValue(stubMutation as any);
  });

  // -------------------------------------------------------------------------
  describe("loading state", () => {
    it("shows a spinner while loading", () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      mockUseRoles.mockReturnValue({ data: undefined, isLoading: true } as any);

      renderPage();

      expect(
        screen.getByRole("status", { name: /loading roles/i })
      ).toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("data display", () => {
    beforeEach(() => {
      mockUseRoles.mockReturnValue({
        data: roles,
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);
    });

    it("renders role names", () => {
      renderPage();

      expect(screen.getByText("admin")).toBeInTheDocument();
      expect(screen.getByText("viewer")).toBeInTheDocument();
    });

    it("shows a System badge for system roles", () => {
      renderPage();

      expect(screen.getByText("System")).toBeInTheDocument();
    });

    it("shows the total role count", () => {
      renderPage();

      expect(screen.getByText("2 roles")).toBeInTheDocument();
    });

    it("shows permission counts", () => {
      renderPage();

      expect(screen.getByText("2 permissions")).toBeInTheDocument();
      expect(screen.getByText("1 permission")).toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("permission-gated elements", () => {
    beforeEach(() => {
      mockUseRoles.mockReturnValue({
        data: [],
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);
    });

    it("shows 'New Role' button with roles:write", () => {
      renderPage(makeAuthCtx((r, a) => r === "roles" && a === "write"));

      expect(
        screen.getByRole("button", { name: /new role/i })
      ).toBeInTheDocument();
    });

    it("hides 'New Role' button without roles:write", () => {
      renderPage(makeAuthCtx(() => false));

      expect(
        screen.queryByRole("button", { name: /new role/i })
      ).not.toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("delete flow", () => {
    beforeEach(() => {
      mockUseRoles.mockReturnValue({
        data: roles,
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);
    });

    it("shows delete button only for non-system roles when user has roles:delete", () => {
      renderPage(makeAuthCtx((r, a) => r === "roles" && a === "delete"));

      // viewer (non-system) → delete button present
      expect(
        screen.getByRole("button", { name: /delete viewer/i })
      ).toBeInTheDocument();
      // admin (system) → no delete button
      expect(
        screen.queryByRole("button", { name: /delete admin/i })
      ).not.toBeInTheDocument();
    });

    it("opens confirmation dialog when delete button is clicked", async () => {
      renderPage(makeAuthCtx((r, a) => r === "roles" && a === "delete"));
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /delete viewer/i }));

      expect(screen.getByRole("dialog")).toBeInTheDocument();
      expect(screen.getByText(/are you sure/i)).toBeInTheDocument();
    });

    it("calls deleteRole.mutateAsync when confirmed", async () => {
      renderPage(makeAuthCtx((r, a) => r === "roles" && a === "delete"));
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /delete viewer/i }));
      await user.click(
        screen.getByRole("button", { name: /^delete$/i })
      );

      await waitFor(() =>
        expect(stubMutation.mutateAsync).toHaveBeenCalledWith("role-2")
      );
    });

    it("shows an API error inside the dialog when deletion fails", async () => {
      const errorMutation = {
        ...stubMutation,
        mutateAsync: vi
          .fn()
          .mockRejectedValue(
            new ApiError(409, "CONFLICT", "Role is assigned to users.")
          ),
        error: new ApiError(409, "CONFLICT", "Role is assigned to users."),
        isError: true,
      };

      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      mockUseDeleteRole.mockReturnValue(errorMutation as any);

      renderPage(makeAuthCtx((r, a) => r === "roles" && a === "delete"));
      const user = userEvent.setup();

      await user.click(screen.getByRole("button", { name: /delete viewer/i }));

      expect(screen.getByRole("alert")).toHaveTextContent(
        "Role is assigned to users."
      );
    });
  });
});
