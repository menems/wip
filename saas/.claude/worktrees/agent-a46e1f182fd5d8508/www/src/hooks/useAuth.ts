import { useMutation, useQuery } from '@connectrpc/connect-query'
import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { AuthService } from '../gen/auth_pb'
import { useAuthContext } from '../context/AuthProvider'

export function useLogin() {
  const { setToken } = useAuthContext()
  return useMutation(AuthService.method.login, {
    onSuccess(data) {
      setToken(data.token)
    },
  })
}

export function useLogout() {
  const { setToken } = useAuthContext()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  return () => {
    setToken(null)
    queryClient.clear()
    void navigate({ to: '/login' })
  }
}

export function useCurrentUser() {
  const { token } = useAuthContext()
  return useQuery(AuthService.method.getCurrentUser, {}, { enabled: token !== null })
}
