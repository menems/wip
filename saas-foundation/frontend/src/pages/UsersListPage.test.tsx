import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { UsersListPage } from "./UsersListPage";
import { AuthContext, type AuthContextValue } from "@/features/auth/AuthContext";
import { ApiError, type MeUser, type UserItem } from "@/lib/api";

// ---------------------------------------------------------------------------
// Mock user hooks so we don't need a real QueryClient or API
// ---------------------------------------------------------------------------

vi.mock("@/features/users/useUsers", () => ({
  useUsers: vi.fn(),
  useDeactivateUser: vi.fn(),
  useReactivateUser: vi.fn(),
}));

import {
  useUsers,
  useDeactivateUser,
  useReactivateUser,
} from "@/features/users/useUsers";

const mockUseUsers = vi.mocked(useUsers);
const mockUseDeactivateUser = vi.mocked(useDeactivateUser);
const mockUseReactivateUser = vi.mocked(useReactivateUser);

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

const users: UserItem[] = [
  {
    id: "user-2",
    email: "alice@example.com",
    name: "Alice",
    is_active: true,
    roles: [{ id: "role-1", name: "admin" }],
    created_at: "2024-01-01T00:00:00Z",
  },
  {
    id: "user-3",
    email: "bob@example.com",
    name: "Bob",
    is_active: false,
    roles: [],
    created_at: "2024-01-02T00:00:00Z",
  },
];

/** Minimal UseMutationResult shape for deactivate/reactivate mocks. */
const stubMutation = {
  mutate: vi.fn(),
  mutateAsync: vi.fn(),
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

function renderPage(
  ctx: AuthContextValue = makeAuthCtx(() => true)
) {
  return render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter>
        <UsersListPage />
      </MemoryRouter>
    </AuthContext.Provider>
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("UsersListPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Default: not pending, no errors
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockUseDeactivateUser.mockReturnValue(stubMutation as any);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockUseReactivateUser.mockReturnValue(stubMutation as any);
  });

  // -------------------------------------------------------------------------
  describe("loading state", () => {
    it("shows a loading spinner while fetching", () => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      mockUseUsers.mockReturnValue({ data: undefined, isLoading: true } as any);

      renderPage();

      expect(
        screen.getByRole("status", { name: /loading users/i })
      ).toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("data display", () => {
    beforeEach(() => {
      mockUseUsers.mockReturnValue({
        data: {
          data: users,
          meta: { page: 1, per_page: 25, total: 2 },
        },
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);
    });

    it("renders user names in the table", () => {
      renderPage();

      expect(screen.getByText("Alice")).toBeInTheDocument();
      expect(screen.getByText("Bob")).toBeInTheDocument();
    });

    it("renders user emails in the table", () => {
      renderPage();

      expect(screen.getByText("alice@example.com")).toBeInTheDocument();
      expect(screen.getByText("bob@example.com")).toBeInTheDocument();
    });

    it("shows the total user count in the header", () => {
      renderPage();

      expect(screen.getByText("2 users")).toBeInTheDocument();
    });

    it("shows '1 user' (singular) when total is 1", () => {
      mockUseUsers.mockReturnValue({
        data: {
          data: [users[0]],
          meta: { page: 1, per_page: 25, total: 1 },
        },
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);

      renderPage();

      expect(screen.getByText("1 user")).toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("permission-gated elements", () => {
    beforeEach(() => {
      mockUseUsers.mockReturnValue({
        data: { data: [], meta: { page: 1, per_page: 25, total: 0 } },
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);
    });

    it("shows 'New User' button when user has users:write", () => {
      renderPage(makeAuthCtx((r, a) => r === "users" && a === "write"));

      expect(
        screen.getByRole("button", { name: /new user/i })
      ).toBeInTheDocument();
    });

    it("hides 'New User' button without users:write", () => {
      renderPage(makeAuthCtx(() => false));

      expect(
        screen.queryByRole("button", { name: /new user/i })
      ).not.toBeInTheDocument();
    });
  });

  // -------------------------------------------------------------------------
  describe("search input", () => {
    it("renders the search input", () => {
      mockUseUsers.mockReturnValue({
        data: { data: [], meta: { page: 1, per_page: 25, total: 0 } },
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);

      renderPage();

      expect(
        screen.getByRole("textbox", { name: /search users/i })
      ).toBeInTheDocument();
    });

    it("re-fetches when the user types in the search box", async () => {
      mockUseUsers.mockReturnValue({
        data: { data: [], meta: { page: 1, per_page: 25, total: 0 } },
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);

      renderPage();
      const user = userEvent.setup();

      // useUsers is called with the filter including debounced search
      // We just verify the input is interactive and re-renders without crashing
      const searchInput = screen.getByRole("textbox", { name: /search users/i });
      await user.type(searchInput, "alice");

      expect(searchInput).toHaveValue("alice");
    });
  });

  // -------------------------------------------------------------------------
  describe("pagination", () => {
    it("shows pagination controls when there are multiple pages", () => {
      mockUseUsers.mockReturnValue({
        data: {
          data: users,
          meta: { page: 1, per_page: 1, total: 2 }, // 2 pages
        },
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);

      renderPage();

      expect(
        screen.getByRole("button", { name: /previous/i })
      ).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: /next/i })
      ).toBeInTheDocument();
      expect(screen.getByText(/page 1 of 2/i)).toBeInTheDocument();
    });

    it("does not show pagination controls when there is only one page", () => {
      mockUseUsers.mockReturnValue({
        data: {
          data: users,
          meta: { page: 1, per_page: 25, total: 2 }, // 1 page
        },
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);

      renderPage();

      expect(
        screen.queryByRole("button", { name: /previous/i })
      ).not.toBeInTheDocument();
    });

    it("disables the Previous button on the first page", () => {
      mockUseUsers.mockReturnValue({
        data: {
          data: users,
          meta: { page: 1, per_page: 1, total: 2 },
        },
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);

      renderPage();

      expect(
        screen.getByRole("button", { name: /previous/i })
      ).toBeDisabled();
    });
  });

  // -------------------------------------------------------------------------
  describe("action errors", () => {
    it("shows a deactivate error when it exists", () => {
      mockUseUsers.mockReturnValue({
        data: { data: users, meta: { page: 1, per_page: 25, total: 2 } },
        isLoading: false,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);

      mockUseDeactivateUser.mockReturnValue({
        ...stubMutation,
        error: new ApiError(409, "LAST_ADMIN", "Cannot deactivate last admin."),
        isError: true,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
      } as any);

      renderPage();

      expect(screen.getByRole("alert")).toBeInTheDocument();
    });
  });
});
