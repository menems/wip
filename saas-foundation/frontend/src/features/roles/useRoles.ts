import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  rolesApi,
  type RoleItem,
  type CreateRoleBody,
  type UpdateRoleBody,
} from "@/lib/api";

/** Stable base query key for all role queries. */
export const ROLES_KEY = ["roles"] as const;

/**
 * useRoles fetches all roles including their permissions.
 * Requires `roles:read` permission.
 */
export function useRoles() {
  return useQuery({
    queryKey: ROLES_KEY,
    queryFn: () => rolesApi.list(),
    select: (data): RoleItem[] => data.data,
  });
}

/**
 * useRole fetches a single role by ID including its permissions.
 * Requires `roles:read` permission.
 */
export function useRole(id: string) {
  return useQuery({
    queryKey: [...ROLES_KEY, id],
    queryFn: () => rolesApi.get(id),
    enabled: id.length > 0,
  });
}

/**
 * useCreateRole returns a mutation that creates a new role.
 * On success, invalidates the roles list cache.
 */
export function useCreateRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateRoleBody) => rolesApi.create(body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ROLES_KEY }),
  });
}

/**
 * useUpdateRole returns a mutation that replaces a role's name, description,
 * and full permission set. On success, invalidates both the list and the
 * individual role cache entry.
 */
export function useUpdateRole(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateRoleBody) => rolesApi.update(id, body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ROLES_KEY });
      void qc.invalidateQueries({ queryKey: [...ROLES_KEY, id] });
    },
  });
}

/**
 * useDeleteRole returns a mutation that deletes a role.
 * The backend blocks deletion of system roles and roles assigned to users.
 * On success, invalidates the roles list cache.
 */
export function useDeleteRole() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => rolesApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ROLES_KEY }),
  });
}
