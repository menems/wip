import { useMutation, useQuery, createConnectQueryKey } from '@connectrpc/connect-query'
import { useQueryClient } from '@tanstack/react-query'
import { UserService } from '../gen/user_pb'

export function useUsers() {
  return useQuery(UserService.method.listUsers)
}

export function useUser(id: string | undefined) {
  return useQuery(UserService.method.getUser, { id: id ?? '' }, { enabled: !!id })
}

export function useCreateUser() {
  const queryClient = useQueryClient()
  return useMutation(UserService.method.createUser, {
    onSuccess() {
      queryClient.invalidateQueries({ queryKey: createConnectQueryKey({ schema: UserService.method.listUsers, cardinality: undefined }) })
    },
  })
}

export function useUpdateUser() {
  const queryClient = useQueryClient()
  return useMutation(UserService.method.updateUser, {
    onSuccess() {
      queryClient.invalidateQueries({ queryKey: createConnectQueryKey({ schema: UserService.method.listUsers, cardinality: undefined }) })
    },
  })
}

export function useDeleteUser() {
  const queryClient = useQueryClient()
  return useMutation(UserService.method.deleteUser, {
    onSuccess() {
      queryClient.invalidateQueries({ queryKey: createConnectQueryKey({ schema: UserService.method.listUsers, cardinality: undefined }) })
    },
  })
}
