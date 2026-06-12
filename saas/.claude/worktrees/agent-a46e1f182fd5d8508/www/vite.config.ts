import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'

export default defineConfig({
  plugins: [
    TanStackRouterVite({ autoCodeSplitting: true, routeFileIgnorePattern: '\\.test\\.(tsx|ts)$' }),
    react(),
    tailwindcss(),
  ],
  server: {
    proxy: {
      '/api.v1.': 'http://localhost:8080',
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
  },
})
