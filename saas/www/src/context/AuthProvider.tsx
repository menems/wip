import { createContext, useCallback, useContext, useMemo, useRef, useState } from 'react'
import { createConnectTransport } from '@connectrpc/connect-web'
import { TransportProvider } from '@connectrpc/connect-query'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

export interface AuthContextValue {
  token: string | null
  setToken: (token: string | null) => void
}

export const AuthContext = createContext<AuthContextValue | null>(null)

const queryClient = new QueryClient()

const TOKEN_KEY = 'auth_token'

interface AuthProviderProps {
  children: React.ReactNode
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [token, setTokenState] = useState<string | null>(
    () => localStorage.getItem(TOKEN_KEY),
  )

  const setToken = useCallback((next: string | null) => {
    if (next === null) {
      localStorage.removeItem(TOKEN_KEY)
    } else {
      localStorage.setItem(TOKEN_KEY, next)
    }
    setTokenState(next)
  }, [])
  const tokenRef = useRef<string | null>(null)
  tokenRef.current = token

  const transport = useMemo(
    () =>
      createConnectTransport({
        baseUrl: '/',
        interceptors: [
          (next) => async (req) => {
            const t = tokenRef.current
            if (t) {
              req.header.set('Authorization', `Bearer ${t}`)
            }
            return next(req)
          },
        ],
      }),
    [],
  )

  return (
    <AuthContext.Provider value={{ token, setToken }}>
      <QueryClientProvider client={queryClient}>
        <TransportProvider transport={transport}>{children}</TransportProvider>
      </QueryClientProvider>
    </AuthContext.Provider>
  )
}

export function useAuthContext(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuthContext must be used within AuthProvider')
  return ctx
}
