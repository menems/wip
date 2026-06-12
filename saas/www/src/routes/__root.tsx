import { createRootRoute, Outlet } from '@tanstack/react-router'
import { ThemeProvider } from '../context/ThemeProvider'

export const Route = createRootRoute({
  component: function Root() {
    return (
      <ThemeProvider>
        <Outlet />
      </ThemeProvider>
    )
  },
})
