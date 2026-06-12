import { renderHook, act, waitFor } from '@testing-library/react'
import { createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthContext } from '../context/AuthProvider'
import { useLogin, useLogout, useCurrentUser } from './useAuth'
import { AuthService } from '../gen/auth_pb'

const mockNavigate = vi.fn()
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useNavigate: () => mockNavigate }
})

function makeWrapper(
  token: string | null,
  setToken: (t: string | null) => void,
  transport: ReturnType<typeof createRouterTransport>,
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <AuthContext.Provider value={{ token, setToken }}>
        <QueryClientProvider client={queryClient}>
          <TransportProvider transport={transport}>{children}</TransportProvider>
        </QueryClientProvider>
      </AuthContext.Provider>
    )
  }
}

describe('useLogin', () => {
  it('calls setToken with the returned JWT on success', async () => {
    const setToken = vi.fn()
    const transport = createRouterTransport(({ service }) => {
      service(AuthService, {
        login: () => ({ token: 'test-jwt' }),
      })
    })
    const { result } = renderHook(() => useLogin(), {
      wrapper: makeWrapper(null, setToken, transport),
    })

    act(() => {
      result.current.mutate({ email: 'admin@localhost', password: 'password' })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(setToken).toHaveBeenCalledWith('test-jwt')
  })

  it('reports error on failed login', async () => {
    const setToken = vi.fn()
    const transport = createRouterTransport(({ service }) => {
      service(AuthService, {
        login: () => {
          throw new Error('invalid credentials')
        },
      })
    })
    const { result } = renderHook(() => useLogin(), {
      wrapper: makeWrapper(null, setToken, transport),
    })

    act(() => {
      result.current.mutate({ email: 'bad@example.com', password: 'wrong' })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(setToken).not.toHaveBeenCalled()
  })
})

describe('useLogout', () => {
  it('calls setToken(null), clears query cache, and navigates to /login', () => {
    const setToken = vi.fn()
    const transport = createRouterTransport(() => {})
    const { result } = renderHook(() => useLogout(), {
      wrapper: makeWrapper('existing-token', setToken, transport),
    })

    act(() => {
      result.current()
    })

    expect(setToken).toHaveBeenCalledWith(null)
    expect(mockNavigate).toHaveBeenCalledWith({ to: '/login' })
  })
})

describe('useCurrentUser', () => {
  it('fetches current user when token is present', async () => {
    const transport = createRouterTransport(({ service }) => {
      service(AuthService, {
        getCurrentUser: () => ({
          userId: 'user-1',
          email: 'admin@localhost',
          role: 'admin',
        }),
      })
    })
    const { result } = renderHook(() => useCurrentUser(), {
      wrapper: makeWrapper('valid-token', vi.fn(), transport),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.email).toBe('admin@localhost')
  })

  it('does not fetch when token is null', () => {
    const transport = createRouterTransport(({ service }) => {
      service(AuthService, {
        getCurrentUser: () => {
          throw new Error('should not be called')
        },
      })
    })
    const { result } = renderHook(() => useCurrentUser(), {
      wrapper: makeWrapper(null, vi.fn(), transport),
    })

    expect(result.current.fetchStatus).toBe('idle')
    expect(result.current.data).toBeUndefined()
  })
})
