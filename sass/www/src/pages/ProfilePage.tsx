import { useState, useEffect } from 'react'
import { useRouter } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getMe, updateMe, clearToken } from '../api/client'

function Avatar({ name, avatarUrl }: { name: string; avatarUrl: string }) {
  if (avatarUrl) {
    return (
      <img
        src={avatarUrl}
        alt={name}
        className="w-20 h-20 rounded-full object-cover ring-4 ring-white shadow"
      />
    )
  }
  const initials = name
    ? name.split(' ').map((n) => n[0]).join('').slice(0, 2).toUpperCase()
    : '?'
  return (
    <div className="w-20 h-20 rounded-full bg-indigo-600 flex items-center justify-center ring-4 ring-white shadow">
      <span className="text-2xl font-bold text-white">{initials}</span>
    </div>
  )
}

export function ProfilePage() {
  const router = useRouter()
  const queryClient = useQueryClient()

  const { data: user, isLoading } = useQuery({
    queryKey: ['me'],
    queryFn: getMe,
  })

  const [name, setName] = useState('')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (user) {
      setName(user.name)
      setAvatarUrl(user.avatar_url)
    }
  }, [user])

  const updateMutation = useMutation({
    mutationFn: () => updateMe({ name, avatar_url: avatarUrl }),
    onSuccess: (updated) => {
      queryClient.setQueryData(['me'], updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    },
  })

  const handleLogout = () => {
    clearToken()
    router.navigate({ to: '/login' })
  }

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault()
    updateMutation.mutate()
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Navbar */}
      <nav className="bg-white border-b border-gray-200">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <span className="text-lg font-semibold text-gray-900">SaaS App</span>
            <button
              onClick={handleLogout}
              className="text-sm text-gray-500 hover:text-gray-900 font-medium transition-colors"
            >
              Logout
            </button>
          </div>
        </div>
      </nav>

      <main className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-10 space-y-6">
        {isLoading ? (
          <div className="space-y-6">
            <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8 animate-pulse">
              <div className="flex items-center gap-5 mb-8">
                <div className="w-20 h-20 rounded-full bg-gray-200" />
                <div className="space-y-2">
                  <div className="h-5 w-32 bg-gray-200 rounded" />
                  <div className="h-4 w-48 bg-gray-200 rounded" />
                </div>
              </div>
              <div className="space-y-4">
                <div className="h-10 bg-gray-200 rounded-lg" />
                <div className="h-10 bg-gray-200 rounded-lg" />
                <div className="h-10 bg-gray-200 rounded-lg" />
              </div>
            </div>
          </div>
        ) : user ? (
          <>
            {/* Profile Card */}
            <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8">
              <div className="flex items-center gap-5 mb-8">
                <Avatar name={name} avatarUrl={avatarUrl} />
                <div>
                  <h2 className="text-xl font-semibold text-gray-900">{user.name || 'No name set'}</h2>
                  <p className="text-gray-500 text-sm">{user.email}</p>
                </div>
              </div>

              <h3 className="text-lg font-semibold text-gray-900 mb-5">Your Profile</h3>

              <form onSubmit={handleSave} className="space-y-5">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Email</label>
                  <input
                    type="email"
                    value={user.email}
                    readOnly
                    className="w-full rounded-lg border border-gray-200 bg-gray-50 px-3.5 py-2.5 text-gray-500 cursor-not-allowed"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Name</label>
                  <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                    placeholder="Your name"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1.5">Avatar URL</label>
                  <input
                    type="url"
                    value={avatarUrl}
                    onChange={(e) => setAvatarUrl(e.target.value)}
                    className="w-full rounded-lg border border-gray-300 px-3.5 py-2.5 text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                    placeholder="https://example.com/avatar.jpg"
                  />
                </div>

                {updateMutation.isError && (
                  <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
                    {(updateMutation.error as Error).message}
                  </p>
                )}

                {saved && (
                  <p className="text-sm text-green-700 bg-green-50 border border-green-200 rounded-lg px-3 py-2">
                    Changes saved successfully.
                  </p>
                )}

                <div className="flex justify-end">
                  <button
                    type="submit"
                    disabled={updateMutation.isPending}
                    className="bg-indigo-600 text-white font-medium py-2.5 px-6 rounded-lg hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    {updateMutation.isPending ? 'Saving...' : 'Save changes'}
                  </button>
                </div>
              </form>
            </div>

            {/* Account Card */}
            <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8">
              <h3 className="text-lg font-semibold text-gray-900 mb-5">Account</h3>
              <dl className="space-y-4">
                <div className="flex justify-between items-center py-3 border-b border-gray-100">
                  <dt className="text-sm font-medium text-gray-500">Member since</dt>
                  <dd className="text-sm text-gray-900">
                    {new Date(user.created_at).toLocaleDateString('en-US', {
                      year: 'numeric',
                      month: 'long',
                      day: 'numeric',
                    })}
                  </dd>
                </div>
                <div className="flex justify-between items-center py-3">
                  <dt className="text-sm font-medium text-gray-500">Account ID</dt>
                  <dd className="text-sm text-gray-900 font-mono">{user.id.slice(0, 8)}...</dd>
                </div>
              </dl>
            </div>
          </>
        ) : null}
      </main>
    </div>
  )
}
