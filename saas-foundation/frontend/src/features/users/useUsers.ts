import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  usersApi,
  type UserFilter,
  type CreateUserBody,
  type UpdateUserBody,
} from "@/lib/api";

/** Stable base query key for all user queries. */
export const USERS_KEY = ["users"] as const;

/**
 * useUsers returns a paginated, searchable list of users.
 * Requires `users:read` permission.
 */
export function useUsers(filter: UserFilter = {}) {
  return useQuery({
    queryKey: [...USERS_KEY, filter],
    queryFn: () => usersApi.list(filter),
  });
}

/**
 * useUser fetches a single user by ID.
 * Requires `users:read` permission.
 */
export function useUser(id: string) {
  return useQuery({
    queryKey: [...USERS_KEY, id],
    queryFn: () => usersApi.get(id),
    enabled: id.length > 0,
  });
}

/**
 * useCreateUser returns a mutation that creates a new user.
 * On success, invalidates the users list cache.
 */
export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateUserBody) => usersApi.create(body),
    onSuccess: () => qc.invalidateQueries({ queryKey: USERS_KEY }),
  });
}

/**
 * useUpdateUser returns a mutation that updates a user's name, email, and role.
 * On success, invalidates both the list and the individual user cache entry.
 */
export function useUpdateUser(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: UpdateUserBody) => usersApi.update(id, body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: USERS_KEY });
      void qc.invalidateQueries({ queryKey: [...USERS_KEY, id] });
    },
  });
}

/**
 * useDeactivateUser returns a mutation that sets is_active=false.
 * Returns a CONFLICT error if the user is the last active admin.
 */
export function useDeactivateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => usersApi.deactivate(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: USERS_KEY }),
  });
}

/**
 * useReactivateUser returns a mutation that sets is_active=true.
 */
export function useReactivateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => usersApi.reactivate(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: USERS_KEY }),
  });
}

/**
 * useResetPassword returns a mutation that sets a new password for a user.
 * This is an admin-initiated reset — no email is sent.
 */
export function useResetPassword() {
  return useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) =>
      usersApi.resetPassword(id, password),
  });
}
