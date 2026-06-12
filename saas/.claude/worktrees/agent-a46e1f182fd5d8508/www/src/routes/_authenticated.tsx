import { createFileRoute, Outlet, Navigate, Link } from '@tanstack/react-router'
import { useAuthContext } from '../context/AuthProvider'
import { useLogout } from '../hooks/useAuth'
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

export const Route = createFileRoute('/_authenticated')({
  component: AuthenticatedLayout,
})

export function AuthenticatedLayout() {
  const { token } = useAuthContext()
  const logout = useLogout()

  if (!token) {
    return <Navigate to="/login" />
  }

  return (
    <div>
      <nav className="border-b bg-background">
        <div className="mx-auto flex max-w-5xl items-center gap-6 px-6 py-3">
          <Link
            to="/"
            className="text-sm font-medium text-muted-foreground hover:text-foreground [&.active]:text-foreground"
          >
            Home
          </Link>
          <Link
            to="/users"
            className="text-sm font-medium text-muted-foreground hover:text-foreground [&.active]:text-foreground"
          >
            Users
          </Link>
          <button
            onClick={logout}
            className="ml-auto text-sm font-medium text-muted-foreground hover:text-foreground"
          >
            Logout
          </button>
          <ThemeToggle />
        </div>
      </nav>
      <Outlet />
    </div>
  )
}
