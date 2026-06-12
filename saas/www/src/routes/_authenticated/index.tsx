import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/')({
  component: function Index() {
    return <main className="flex min-h-screen items-center justify-center" />
  },
})
