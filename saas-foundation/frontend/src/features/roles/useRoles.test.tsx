import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { type ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// ---------------------------------------------------------------------------
// Mock the API module
// ---------------------------------------------------------------------------

vi.mock("@/lib/api", () => ({
  rolesApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}));

import { rolesApi } from "@/lib/api";
import {
  useRoles,
  useRole,
  useCreateRole,
  useUpdateRole,
  useDeleteRole,
} from "./useRoles";

const mockList = vi.mocked(rolesApi.list);
const mockGet = vi.mocked(rolesApi.get);
const mockCreate = vi.mocked(rolesApi.create);
const mockUpdate = vi.mocked(rolesApi.update);
const mockDelete = vi.mocked(rolesApi.delete);

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const adminRole = {
  id: "role-1",
  name: "admin",
  description: "Administrator",
  is_system: true,
  permissions: [
    { resource: "users", action: "read" },
    { resource: "users", action: "write" },
  ],
  created_at: "2024-01-01T00:00:00Z",
};

const viewerRole = {
  id: "role-2",
  name: "viewer",
  description: "Read-only",
  is_system: false,
  permissions: [{ resource: "users", action: "read" }],
  created_at: "2024-01-02T00:00:00Z",
};

// ---------------------------------------------------------------------------
// Wrapper — fresh QueryClient per test
// ---------------------------------------------------------------------------

function makeWrapper() {
  const qc = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("useRoles hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // -------------------------------------------------------------------------
  describe("useRoles", () => {
    it("fetches the roles list and selects the data array", async () => {
      mockList.mockResolvedValue({ data: [adminRole, viewerRole] });

      const { result } = renderHook(() => useRoles(), {
        wrapper: makeWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(mockList).toHaveBeenCalledOnce();
      expect(result.current.data).toEqual([adminRole, viewerRole]);
    });
  });

  // -------------------------------------------------------------------------
  describe("useRole", () => {
    it("fetches a single role by id", async () => {
      mockGet.mockResolvedValue(adminRole);

      const { result } = renderHook(() => useRole("role-1"), {
        wrapper: makeWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));

      expect(mockGet).toHaveBeenCalledWith("role-1");
      expect(result.current.data).toEqual(adminRole);
    });

    it("does not fetch when id is empty", () => {
      const { result } = renderHook(() => useRole(""), {
        wrapper: makeWrapper(),
      });

      expect(result.current.fetchStatus).toBe("idle");
      expect(mockGet).not.toHaveBeenCalled();
    });
  });

  // -------------------------------------------------------------------------
  describe("useCreateRole", () => {
    it("calls rolesApi.create with the supplied body", async () => {
      mockCreate.mockResolvedValue(viewerRole);

      const { result } = renderHook(() => useCreateRole(), {
        wrapper: makeWrapper(),
      });

      await act(async () => {
        const returned = await result.current.mutateAsync({
          name: "viewer",
          description: "Read-only",
          permissions: [{ resource: "users", action: "read" }],
        });
        expect(returned).toEqual(viewerRole);
      });

      expect(mockCreate).toHaveBeenCalledWith({
        name: "viewer",
        description: "Read-only",
        permissions: [{ resource: "users", action: "read" }],
      });
    });
  });

  // -------------------------------------------------------------------------
  describe("useUpdateRole", () => {
    it("calls rolesApi.update with the correct id and body", async () => {
      const updated = { ...viewerRole, name: "viewer-updated" };
      mockUpdate.mockResolvedValue(updated);

      const { result } = renderHook(() => useUpdateRole("role-2"), {
        wrapper: makeWrapper(),
      });

      await act(async () => {
        await result.current.mutateAsync({
          name: "viewer-updated",
          permissions: [{ resource: "users", action: "read" }],
        });
      });

      expect(mockUpdate).toHaveBeenCalledWith("role-2", {
        name: "viewer-updated",
        permissions: [{ resource: "users", action: "read" }],
      });
    });
  });

  // -------------------------------------------------------------------------
  describe("useDeleteRole", () => {
    it("calls rolesApi.delete with the correct id", async () => {
      mockDelete.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteRole(), {
        wrapper: makeWrapper(),
      });

      await act(async () => {
        await result.current.mutateAsync("role-2");
      });

      expect(mockDelete).toHaveBeenCalledWith("role-2");
    });

    it("exposes isError=true when deletion fails", async () => {
      mockDelete.mockRejectedValue(new Error("Role is assigned to users"));

      const { result } = renderHook(() => useDeleteRole(), {
        wrapper: makeWrapper(),
      });

      act(() => {
        result.current.mutate("role-2");
      });

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });
});
