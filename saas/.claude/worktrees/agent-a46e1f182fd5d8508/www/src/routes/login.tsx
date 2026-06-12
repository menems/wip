import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { ConnectError } from '@connectrpc/connect'
import { useLogin } from '../hooks/useAuth'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Label } from '../components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/card'
import { useTheme } from '../context/ThemeProvider'

function MoonIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24"
         fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
    </svg>
  )
}

function SunIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24"
         fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="4"/>
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/>
    </svg>
  )
}

function ThemeToggle() {
  const { theme, toggleTheme } = useTheme()
  return (
    <button
      onClick={toggleTheme}
      aria-label={theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}
      className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground"
    >
      {theme === 'light' ? <MoonIcon /> : <SunIcon />}
    </button>
  )
}

export const Route = createFileRoute('/login')({
  component: LoginPage,
})

export function LoginPage() {
  const navigate = useNavigate()
  const login = useLogin()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [fieldErrors, setFieldErrors] = useState<{ email?: string; password?: string }>({})

  function validate(): { email?: string; password?: string } {
    const errors: { email?: string; password?: string } = {}
    if (!email) {
      errors.email = 'Email is required'
    } else if (!/^[^\s@]+@[^\s@]+$/.test(email)) {
      errors.email = 'Enter a valid email address'
    }
    if (!password) {
      errors.password = 'Password is required'
    }
    return errors
  }

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const errors = validate()
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors)
      return
    }
    setFieldErrors({})
    login.mutate(
      { email, password },
      { onSuccess: () => navigate({ to: '/' }) },
    )
  }

  const serverError =
    login.error instanceof ConnectError
      ? login.error.rawMessage
      : login.error
        ? 'Login failed. Please try again.'
        : null

  return (
    <main className="relative flex min-h-screen items-center justify-center bg-background">
      <div className="absolute top-4 right-4">
        <ThemeToggle />
      </div>
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Sign in</CardTitle>
        </CardHeader>
        <CardContent>
          {serverError && (
            <p
              role="alert"
              className="mb-4 rounded bg-destructive/10 px-3 py-2 text-sm text-destructive"
            >
              {serverError}
            </p>
          )}

          <form onSubmit={handleSubmit} noValidate>
            <div className="mb-4">
              <Label htmlFor="email" className="mb-1 block">
                Email
              </Label>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                aria-describedby={fieldErrors.email ? 'email-error' : undefined}
                aria-invalid={fieldErrors.email ? true : undefined}
                className={fieldErrors.email ? 'border-destructive focus-visible:ring-destructive' : undefined}
              />
              {fieldErrors.email && (
                <p id="email-error" className="mt-1 text-xs text-destructive">
                  {fieldErrors.email}
                </p>
              )}
            </div>

            <div className="mb-6">
              <Label htmlFor="password" className="mb-1 block">
                Password
              </Label>
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                aria-describedby={fieldErrors.password ? 'password-error' : undefined}
                aria-invalid={fieldErrors.password ? true : undefined}
                className={fieldErrors.password ? 'border-destructive focus-visible:ring-destructive' : undefined}
              />
              {fieldErrors.password && (
                <p id="password-error" className="mt-1 text-xs text-destructive">
                  {fieldErrors.password}
                </p>
              )}
            </div>

            <Button type="submit" disabled={login.isPending} className="w-full">
              {login.isPending ? 'Signing in…' : 'Sign in'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  )
}
