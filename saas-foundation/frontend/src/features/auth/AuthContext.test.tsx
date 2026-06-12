import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { type ReactNode } from "react";
import { AuthProvider, useAuth } from "./AuthContext";

// ---------------------------------------------------------------------------
// Mock the API module
// ---------------------------------------------------------------------------

vi.mock("@/lib/api", () => ({
  authApi: {
    me: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
  },
  rolesApi: {
    list: vi.fn(),
  },
}));

import { authApi, rolesApi } from "@/lib/api";

// Typed mock helpers
const mockMe = vi.mocked(authApi.me);
const mockLogin = vi.mocked(authApi.login);
const mockLogout = vi.mocked(authApi.logout);
const mockRolesList = vi.mocked(rolesApi.list);

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const adminUser = {
  id: "user-1",
  email: "admin@example.com",
  name: "Admin",
  is_active: true,
  roles: [{ id: "role-1", name: "admin" }],
  created_at: "2024-01-01T00:00:00Z",
};

const adminRole = {
  id: "role-1",
  name: "admin",
  description: "Administrator",
  is_system: true,
  permissions: [
    { resource: "users", action: "read" },
    { resource: "users", action: "write" },
    { resource: "users", action: "delete" },
    { resource: "roles", action: "read" },
    { resource: "roles", action: "write" },
    { resource: "roles", action: "delete" },
    { resource: "audit_logs", action: "read" },
  ],
  created_at: "2024-01-01T00:00:00Z",
};

const viewerUser = {
  ...adminUser,
  id: "user-2",
  email: "viewer@example.com",
  roles: [{ id: "role-2", name: "viewer" }],
};

const viewerRole = {
  id: "role-2",
  name: "viewer",
  description: "Read-only",
  is_system: false,
  permissions: [
    { resource: "users", action: "read" },
    { resource: "audit_logs", action: "read" },
  ],
  created_at: "2024-01-01T00:00:00Z",
};

// ---------------------------------------------------------------------------
// Wrapper
// ---------------------------------------------------------------------------

function wrapper({ children }: { children: ReactNode }) {
  return <AuthProvider>{children}</AuthProvider>;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("AuthContext", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("initial load", () => {
    it("sets user and permissions after /auth/me + /roles succeed", async () => {
      mockMe.mockResolvedValue({ user: adminUser });
      mockRolesList.mockResolvedValue({ data: [adminRole, viewerRole] });

      const { result } = renderHook(() => useAuth(), { wrapper });

      expect(result.current.isLoading).toBe(true);

      await waitFor(() => expect(result.current.isLoading).toBe(false));

      expect(result.current.user).toEqual(adminUser);
      expect(result.current.hasPermission("users", "read")).toBe(true);
      expect(result.current.hasPermission("roles", "delete")).toBe(true);
    });

    it("sets user to null when /auth/me returns 401", async () => {
      mockMe.mockRejectedValue(new Error("Unauthorized"));

      const { result } = renderHook(() => useAuth(), { wrapper });

      await waitFor(() => expect(result.current.isLoading).toBe(false));

      expect(result.current.user).toBeNull();
      expect(result.current.hasPermission("users", "read")).toBe(false);
    });

    it("still sets the user when /roles fails but /auth/me succeeds", async () => {
      mockMe.mockResolvedValue({ user: adminUser });
      mockRolesList.mockRejectedValue(new Error("Forbidden"));

      const { result } = renderHook(() => useAuth(), { wrapper });

      await waitFor(() => expect(result.current.isLoading).toBe(false));

      expect(result.current.user).toEqual(adminUser);
      // No permissions loaded — fail gracefully
      expect(result.current.hasPermission("users", "read")).toBe(false);
    });
  });

  describe("hasPermission", () => {
    beforeEach(() => {
      mockMe.mockResolvedValue({ user: viewerUser });
      mockRolesList.mockResolvedValue({ data: [adminRole, viewerRole] });
    });

    it("returns true for a permission the user has", async () => {
      const { result } = renderHook(() => useAuth(), { wrapper });
      await waitFor(() => expect(result.current.isLoading).toBe(false));

      expect(result.current.hasPermission("users", "read")).toBe(true);
      expect(result.current.hasPermission("audit_logs", "read")).toBe(true);
    });

    it("returns false for a permission the user does not have", async () => {
      const { result } = renderHook(() => useAuth(), { wrapper });
      await waitFor(() => expect(result.current.isLoading).toBe(false));

      expect(result.current.hasPermission("users", "write")).toBe(false);
      expect(result.current.hasPermission("roles", "read")).toBe(false);
    });
  });

  describe("login", () => {
    it("fetches user profile after successful login", async () => {
      // First call from mount (unauthenticated), second after login
      mockMe
        .mockRejectedValueOnce(new Error("not logged in"))
        .mockResolvedValueOnce({ user: adminUser });
      mockRolesList.mockResolvedValue({ data: [adminRole] });
      mockLogin.mockResolvedValue({
        user: { id: "user-1", email: "admin@example.com", name: "Admin", roles: ["admin"] },
      });

      const { result } = renderHook(() => useAuth(), { wrapper });
      await waitFor(() => expect(result.current.isLoading).toBe(false));

      expect(result.current.user).toBeNull();

      await act(async () => {
        await result.current.login("admin@example.com", "changeme");
      });

      expect(result.current.user).toEqual(adminUser);
      expect(result.current.hasPermission("users", "read")).toBe(true);
    });
  });

  describe("logout", () => {
    it("clears user and permissions", async () => {
      mockMe.mockResolvedValue({ user: adminUser });
      mockRolesList.mockResolvedValue({ data: [adminRole] });
      mockLogout.mockResolvedValue(undefined);

      const { result } = renderHook(() => useAuth(), { wrapper });
      await waitFor(() => expect(result.current.isLoading).toBe(false));
      expect(result.current.user).toEqual(adminUser);

      await act(async () => {
        await result.current.logout();
      });

      expect(result.current.user).toBeNull();
      expect(result.current.hasPermission("users", "read")).toBe(false);
    });

    it("still clears state when the API call fails", async () => {
      mockMe.mockResolvedValue({ user: adminUser });
      mockRolesList.mockResolvedValue({ data: [adminRole] });
      mockLogout.mockRejectedValue(new Error("Network error"));

      const { result } = renderHook(() => useAuth(), { wrapper });
      await waitFor(() => expect(result.current.isLoading).toBe(false));

      await act(async () => {
        await result.current.logout();
      });

      expect(result.current.user).toBeNull();
    });
  });

  describe("auth:unauthenticated event", () => {
    it("clears user when the event fires", async () => {
      mockMe.mockResolvedValue({ user: adminUser });
      mockRolesList.mockResolvedValue({ data: [adminRole] });

      const { result } = renderHook(() => useAuth(), { wrapper });
      await waitFor(() => expect(result.current.isLoading).toBe(false));
      expect(result.current.user).toEqual(adminUser);

      act(() => {
        window.dispatchEvent(new CustomEvent("auth:unauthenticated"));
      });

      expect(result.current.user).toBeNull();
      expect(result.current.hasPermission("users", "read")).toBe(false);
    });
  });
});
