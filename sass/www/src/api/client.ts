export interface User {
  id: string
  email: string
  name: string
  avatar_url: string
  created_at: string
  updated_at: string
}

export interface AuthResponse {
  token: string
  user: User
}

export function getToken(): string | null {
  return localStorage.getItem('auth_token')
}

export function setToken(token: string): void {
  localStorage.setItem('auth_token', token)
}

export function clearToken(): void {
  localStorage.removeItem('auth_token')
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init?.headers as Record<string, string>),
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const res = await fetch(`/api${path}`, { ...init, headers })
  const data = await res.json()

  if (!res.ok) {
    throw new Error(data.error ?? `Request failed: ${res.status}`)
  }
  return data as T
}

export function login(email: string, password: string): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

export function register(email: string, password: string, name: string): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, password, name }),
  })
}

export function getMe(): Promise<User> {
  return apiFetch<User>('/users/me')
}

export function updateMe(data: { name: string; avatar_url: string }): Promise<User> {
  return apiFetch<User>('/users/me', {
    method: 'PATCH',
    body: JSON.stringify(data),
  })
}
