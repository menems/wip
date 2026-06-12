import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import type { User } from '../../gen/user_pb'
import { Role } from '../../gen/user_pb'
import { useUsers } from '../../hooks/useUsers'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../../components/ui/table'
import { Badge } from '../../components/ui/badge'
import { Button } from '../../components/ui/button'
import { CreateUserDialog } from '../../components/CreateUserDialog'
import { EditUserDialog } from '../../components/EditUserDialog'
import { DeleteUserDialog } from '../../components/DeleteUserDialog'

export const Route = createFileRoute('/_authenticated/users')({
  component: UsersPage,
})

function roleBadge(role: Role) {
  switch (role) {
    case Role.ADMIN:
      return <Badge>Admin</Badge>
    case Role.MEMBER:
      return <Badge variant="secondary">Member</Badge>
    default:
      return <Badge variant="outline">Unknown</Badge>
  }
}

function formatDate(timestamp: { seconds: bigint } | undefined): string {
  if (!timestamp) return '—'
  return new Date(Number(timestamp.seconds) * 1000).toLocaleDateString()
}

function UsersPage() {
  const { data, isLoading, isError } = useUsers()
  const users = data?.users ?? []

  const [createOpen, setCreateOpen] = useState(false)
  const [editUser, setEditUser] = useState<User | null>(null)
  const [deleteUser, setDeleteUser] = useState<User | null>(null)

  return (
    <main className="mx-auto max-w-5xl p-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Users</h1>
        <Button onClick={() => setCreateOpen(true)}>Add user</Button>
      </div>

      {isLoading && (
        <p className="text-muted-foreground">Loading users…</p>
      )}

      {isError && (
        <p className="text-destructive">Failed to load users.</p>
      )}

      {!isLoading && !isError && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Email</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="w-[100px]">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  No users found.
                </TableCell>
              </TableRow>
            ) : (
              users.map((user) => (
                <TableRow key={user.id}>
                  <TableCell className="font-medium">{user.name}</TableCell>
                  <TableCell>{user.email}</TableCell>
                  <TableCell>{roleBadge(user.role)}</TableCell>
                  <TableCell>{formatDate(user.createdAt)}</TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditUser(user)}
                      >
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDeleteUser(user)}
                      >
                        Delete
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      )}

      <CreateUserDialog open={createOpen} onOpenChange={setCreateOpen} />
      <EditUserDialog
        user={editUser}
        open={editUser !== null}
        onOpenChange={(open) => { if (!open) setEditUser(null) }}
      />
      <DeleteUserDialog
        user={deleteUser}
        open={deleteUser !== null}
        onOpenChange={(open) => { if (!open) setDeleteUser(null) }}
      />
    </main>
  )
}
