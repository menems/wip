import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// ---------------------------------------------------------------------------
// Mock the API module
// ---------------------------------------------------------------------------

vi.mock("@/lib/api", () => ({
  usersApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    deactivate: vi.fn(),
    reactivate: vi.fn(),
    resetPassword: vi.fn(),
  },
}));

import { usersApi } from "@/lib/api";
import {
  useUsers,
  useUser,
  useCreateUser,
  useUpdateUser,
  useDeactivateUser,
  useReactivateUser,
  useResetPassword,
} from "./useUsers";

const mockList = vi.mocked(usersApi.list);
const mockGet = vi.mocked(usersApi.get);
const mockCreate = vi.mocked(usersApi.create);
const mockUpdate = vi.mocked(usersApi.update);
const mockDeactivate = vi.mocked(usersApi.deactivate);
const mockReactivate = vi.mocked(usersApi.reactivate);
const mockResetPassword = vi.mocked(usersApi.resetPassword);

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const mockUser = {
  id: "user-1",
  email: "alice@example.com",
  name: "Alice",
  is_active: true,
  roles: [{ id: "role-1", name: "admin" }],
  created_at: "2024-01-01T00:00:00Z",
};

const mockListResponse = {
  data: [mockUser],
  meta: { page: 1, per_page: 25, total: 1 },
};

// ---------------------------------------------------------------------------
// Wrapper — fresh QueryClient per test to avoid cache bleeding
// ---------------------------------------------------------------------------

function makeWrapper() {
  const qc = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    );
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("useUsers hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // -------------------------------------------------------------------------
  describe("useUsers", () => {
    it("calls usersApi.list with the supplied filter and returns the response", async () => {
      mockList.mockResolvedValue(mockListResponse);

      const { result } = renderHook(
        () => useUsers({ page: 1, per_page: 25 }),
        { wrapper: makeWrapper() }
      );

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(mockList).toHaveBeenCalledWith({ page: 1, per_page: 25 });
      expect(result.current.data).toEqual(mockListResponse);
    });

    it("uses an empty filter object by default", async () => {
      mockList.mockResolvedValue(mockListResponse);

      const { result } = renderHook(() => useUsers(), {
        wrapper: makeWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(mockList).toHaveBeenCalledWith({});
    });
  });

  // -------------------------------------------------------------------------
  describe("useUser", () => {
    it("fetches a single user by id", async () => {
      mockGet.mockResolvedValue(mockUser);

      const { result } = renderHook(() => useUser("user-1"), {
        wrapper: makeWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(mockGet).toHaveBeenCalledWith("user-1");
      expect(result.current.data).toEqual(mockUser);
    });

    it("does not fetch when id is empty string", () => {
      const { result } = renderHook(() => useUser(""), {
        wrapper: makeWrapper(),
      });

      // enabled: false → fetchStatus is idle, API never called
      expect(result.current.fetchStatus).toBe("idle");
      expect(mockGet).not.toHaveBeenCalled();
    });
  });

  // -------------------------------------------------------------------------
  describe("useCreateUser", () => {
    it("calls usersApi.create with the supplied body and returns the new user", async () => {
      mockCreate.mockResolvedValue(mockUser);

      const { result } = renderHook(() => useCreateUser(), {
        wrapper: makeWrapper(),
      });

      await act(async () => {
        const returned = await result.current.mutateAsync({
          name: "Alice",
          email: "alice@example.com",
          password: "password123",
          role_id: "role-1",
        });
        expect(returned).toEqual(mockUser);
      });

      expect(mockCreate).toHaveBeenCalledWith({
        name: "Alice",
        email: "alice@example.com",
        password: "password123",
        role_id: "role-1",
      });
    });

    it("exposes isError=true when the API call fails", async () => {
      mockCreate.mockRejectedValue(new Error("Email conflict"));

      const { result } = renderHook(() => useCreateUser(), {
        wrapper: makeWrapper(),
      });

      // Use mutate (not mutateAsync) so we don't need to catch the throw,
      // then waitFor TanStack Query to settle into error state.
      act(() => {
        result.current.mutate({
          name: "Alice",
          email: "alice@example.com",
          password: "password123",
          role_id: "role-1",
        });
      });

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  // -------------------------------------------------------------------------
  describe("useUpdateUser", () => {
    it("calls usersApi.update with the correct id and body", async () => {
      const updated = { ...mockUser, name: "Alice Updated" };
      mockUpdate.mockResolvedValue(updated);

      const { result } = renderHook(() => useUpdateUser("user-1"), {
        wrapper: makeWrapper(),
      });

      await act(async () => {
        const returned = await result.current.mutateAsync({
          name: "Alice Updated",
        });
        expect(returned).toEqual(updated);
      });

      expect(mockUpdate).toHaveBeenCalledWith("user-1", {
        name: "Alice Updated",
      });
    });
  });

  // -------------------------------------------------------------------------
  describe("useDeactivateUser", () => {
    it("calls usersApi.deactivate with the correct id", async () => {
      const deactivated = { ...mockUser, is_active: false };
      mockDeactivate.mockResolvedValue(deactivated);

      const { result } = renderHook(() => useDeactivateUser(), {
        wrapper: makeWrapper(),
      });

      await act(async () => {
        await result.current.mutateAsync("user-1");
      });

      expect(mockDeactivate).toHaveBeenCalledWith("user-1");
    });
  });

  // -------------------------------------------------------------------------
  describe("useReactivateUser", () => {
    it("calls usersApi.reactivate with the correct id", async () => {
      mockReactivate.mockResolvedValue(mockUser);

      const { result } = renderHook(() => useReactivateUser(), {
        wrapper: makeWrapper(),
      });

      await act(async () => {
        await result.current.mutateAsync("user-1");
      });

      expect(mockReactivate).toHaveBeenCalledWith("user-1");
    });
  });

  // -------------------------------------------------------------------------
  describe("useResetPassword", () => {
    it("calls usersApi.resetPassword with the correct id and password", async () => {
      mockResetPassword.mockResolvedValue(undefined);

      const { result } = renderHook(() => useResetPassword(), {
        wrapper: makeWrapper(),
      });

      await act(async () => {
        await result.current.mutateAsync({
          id: "user-1",
          password: "newpassword123",
        });
      });

      expect(mockResetPassword).toHaveBeenCalledWith(
        "user-1",
        "newpassword123"
      );
    });
  });
});
