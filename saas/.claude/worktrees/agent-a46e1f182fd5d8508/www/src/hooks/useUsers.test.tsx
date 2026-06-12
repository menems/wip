import { renderHook, act, waitFor } from '@testing-library/react'
import { createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { UserService } from '../gen/user_pb'
import { Role } from '../gen/user_pb'
import { useUsers, useUser, useCreateUser, useUpdateUser, useDeleteUser } from './useUsers'

function makeWrapper(transport: ReturnType<typeof createRouterTransport>) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <TransportProvider transport={transport}>{children}</TransportProvider>
      </QueryClientProvider>
    )
  }
}

const mockUser = {
  id: 'user-1',
  email: 'alice@example.com',
  name: 'Alice',
  role: Role.ADMIN,
}

const mockUser2 = {
  id: 'user-2',
  email: 'bob@example.com',
  name: 'Bob',
  role: Role.MEMBER,
}

describe('useUsers', () => {
  it('fetches the list of users', async () => {
    const transport = createRouterTransport(({ service }) => {
      service(UserService, {
        listUsers: () => ({ users: [mockUser, mockUser2] }),
      })
    })
    const { result } = renderHook(() => useUsers(), {
      wrapper: makeWrapper(transport),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.users).toHaveLength(2)
    expect(result.current.data?.users[0].email).toBe('alice@example.com')
  })
})

describe('useUser', () => {
  it('fetches a single user by ID', async () => {
    const transport = createRouterTransport(({ service }) => {
      service(UserService, {
        getUser: () => ({ user: mockUser }),
      })
    })
    const { result } = renderHook(() => useUser('user-1'), {
      wrapper: makeWrapper(transport),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.user?.email).toBe('alice@example.com')
  })

  it('does not fetch when id is undefined', () => {
    const transport = createRouterTransport(({ service }) => {
      service(UserService, {
        getUser: () => {
          throw new Error('should not be called')
        },
      })
    })
    const { result } = renderHook(() => useUser(undefined), {
      wrapper: makeWrapper(transport),
    })

    expect(result.current.fetchStatus).toBe('idle')
    expect(result.current.data).toBeUndefined()
  })
})

describe('useCreateUser', () => {
  it('creates a user and reports success', async () => {
    const transport = createRouterTransport(({ service }) => {
      service(UserService, {
        createUser: () => ({ user: mockUser }),
        listUsers: () => ({ users: [] }),
      })
    })
    const { result } = renderHook(() => useCreateUser(), {
      wrapper: makeWrapper(transport),
    })

    act(() => {
      result.current.mutate({
        email: 'alice@example.com',
        name: 'Alice',
        password: 'secret123',
        role: Role.ADMIN,
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.user?.email).toBe('alice@example.com')
  })
})

describe('useUpdateUser', () => {
  it('updates a user and reports success', async () => {
    const updatedUser = { ...mockUser, name: 'Alice Updated' }
    const transport = createRouterTransport(({ service }) => {
      service(UserService, {
        updateUser: () => ({ user: updatedUser }),
        listUsers: () => ({ users: [] }),
      })
    })
    const { result } = renderHook(() => useUpdateUser(), {
      wrapper: makeWrapper(transport),
    })

    act(() => {
      result.current.mutate({
        id: 'user-1',
        email: 'alice@example.com',
        name: 'Alice Updated',
        role: Role.ADMIN,
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.user?.name).toBe('Alice Updated')
  })
})

describe('useDeleteUser', () => {
  it('deletes a user and reports success', async () => {
    const transport = createRouterTransport(({ service }) => {
      service(UserService, {
        deleteUser: () => ({}),
        listUsers: () => ({ users: [] }),
      })
    })
    const { result } = renderHook(() => useDeleteUser(), {
      wrapper: makeWrapper(transport),
    })

    act(() => {
      result.current.mutate({ id: 'user-1' })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
  })
})
