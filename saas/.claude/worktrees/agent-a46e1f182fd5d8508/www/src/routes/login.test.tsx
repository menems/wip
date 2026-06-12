import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthContext } from '../context/AuthProvider'
import { AuthService } from '../gen/auth_pb'
import { LoginPage } from './login'

function makeRouter() {
  const rootRoute = createRootRoute()
  const loginRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/login',
    component: LoginPage,
  })
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => <div>Home page</div>,
  })
  return createRouter({
    routeTree: rootRoute.addChildren([loginRoute, indexRoute]),
    history: createMemoryHistory({ initialEntries: ['/login'] }),
  })
}

interface RenderOptions {
  setToken?: ReturnType<typeof vi.fn>
  transport?: ReturnType<typeof createRouterTransport>
}

function renderLoginPage({ setToken = vi.fn(), transport = createRouterTransport(() => {}) }: RenderOptions = {}) {
  const qc = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  const router = makeRouter()

  render(
    <AuthContext.Provider value={{ token: null, setToken }}>
      <QueryClientProvider client={qc}>
        <TransportProvider transport={transport}>
          <RouterProvider router={router} />
        </TransportProvider>
      </QueryClientProvider>
    </AuthContext.Provider>,
  )

  return { setToken, router }
}

describe('LoginPage', () => {
  it('renders email and password fields and submit button', async () => {
    renderLoginPage()

    expect(await screen.findByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
  })

  it('shows validation errors when submitted with empty fields', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.click(await screen.findByRole('button', { name: /sign in/i }))

    expect(await screen.findByText(/email is required/i)).toBeInTheDocument()
    expect(screen.getByText(/password is required/i)).toBeInTheDocument()
  })

  it('shows validation error for an invalid email format', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(await screen.findByLabelText(/email/i), 'not-an-email')
    await user.type(screen.getByLabelText(/password/i), 'secret')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(await screen.findByText(/valid email address/i)).toBeInTheDocument()
  })

  it('calls the login mutation and navigates to / on success', async () => {
    const user = userEvent.setup()
    const setToken = vi.fn()
    const transport = createRouterTransport(({ service }) => {
      service(AuthService, {
        login: () => ({ token: 'jwt-token' }),
      })
    })

    renderLoginPage({ setToken, transport })

    await user.type(await screen.findByLabelText(/email/i), 'admin@localhost')
    await user.type(screen.getByLabelText(/password/i), 'password')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => expect(screen.getByText('Home page')).toBeInTheDocument())
    expect(setToken).toHaveBeenCalledWith('jwt-token')
  })

  it('shows a server error alert on login failure', async () => {
    const user = userEvent.setup()
    const transport = createRouterTransport(({ service }) => {
      service(AuthService, {
        login: () => {
          throw new Error('invalid credentials')
        },
      })
    })

    renderLoginPage({ transport })

    await user.type(await screen.findByLabelText(/email/i), 'admin@localhost')
    await user.type(screen.getByLabelText(/password/i), 'wrongpassword')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })
})
