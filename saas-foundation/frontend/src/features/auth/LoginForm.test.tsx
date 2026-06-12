import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { LoginForm } from "./LoginForm";
import { AuthContext, type AuthContextValue } from "./AuthContext";
import { ApiError } from "@/lib/api";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const mockNavigate = vi.fn();

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

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

function renderLoginForm(ctx: AuthContextValue) {
  return render(
    <AuthContext.Provider value={ctx}>
      <MemoryRouter>
        <Routes>
          <Route path="/" element={<LoginForm />} />
          <Route path="/dashboard" element={<div>Dashboard</div>} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("LoginForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders email and password fields and a submit button", () => {
    renderLoginForm(makeAuthContext({}));

    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /sign in/i })).toBeInTheDocument();
  });

  it("calls login with entered credentials on submit", async () => {
    const login = vi.fn().mockResolvedValue(undefined);
    renderLoginForm(makeAuthContext({ login }));
    const user = userEvent.setup();

    await user.type(screen.getByLabelText(/email/i), "admin@example.com");
    await user.type(screen.getByLabelText(/password/i), "changeme");
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(login).toHaveBeenCalledWith("admin@example.com", "changeme");
    });
  });

  it("navigates to /dashboard on successful login", async () => {
    const login = vi.fn().mockResolvedValue(undefined);
    renderLoginForm(makeAuthContext({ login }));
    const user = userEvent.setup();

    await user.type(screen.getByLabelText(/email/i), "admin@example.com");
    await user.type(screen.getByLabelText(/password/i), "changeme");
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/dashboard", {
        replace: true,
      });
    });
  });

  it("shows an error message on 401", async () => {
    const login = vi
      .fn()
      .mockRejectedValue(new ApiError(401, "UNAUTHORIZED", "Invalid email or password"));
    renderLoginForm(makeAuthContext({ login }));
    const user = userEvent.setup();

    await user.type(screen.getByLabelText(/email/i), "bad@example.com");
    await user.type(screen.getByLabelText(/password/i), "wrong");
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        /invalid email or password/i
      );
    });
  });

  it("shows deactivated message on ACCOUNT_DEACTIVATED", async () => {
    const login = vi
      .fn()
      .mockRejectedValue(new ApiError(403, "ACCOUNT_DEACTIVATED", "Account deactivated"));
    renderLoginForm(makeAuthContext({ login }));
    const user = userEvent.setup();

    await user.type(screen.getByLabelText(/email/i), "inactive@example.com");
    await user.type(screen.getByLabelText(/password/i), "pass");
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(/deactivated/i);
    });
  });

  it("disables the submit button while submitting", async () => {
    // Never resolves so the button stays disabled
    const login = vi.fn().mockReturnValue(new Promise(() => {}));
    renderLoginForm(makeAuthContext({ login }));
    const user = userEvent.setup();

    await user.type(screen.getByLabelText(/email/i), "admin@example.com");
    await user.type(screen.getByLabelText(/password/i), "pass");
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /signing in/i })).toBeDisabled();
    });
  });
});
