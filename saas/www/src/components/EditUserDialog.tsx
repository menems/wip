import { useEffect, useState, type FormEvent } from 'react'
import type { User } from '../gen/user_pb'
import { Role } from '../gen/user_pb'
import { useUpdateUser } from '../hooks/useUsers'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from './ui/dialog'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Select, SelectOption } from './ui/select'

interface EditUserDialogProps {
  user: User | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface FormErrors {
  name?: string
  email?: string
  role?: string
}

function validate(fields: { name: string; email: string; role: string }): FormErrors {
  const errors: FormErrors = {}

  if (!fields.name.trim()) {
    errors.name = 'Name is required'
  }

  if (!fields.email.trim()) {
    errors.email = 'Email is required'
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(fields.email)) {
    errors.email = 'Invalid email address'
  }

  if (!fields.role) {
    errors.role = 'Role is required'
  }

  return errors
}

export function EditUserDialog({ user, open, onOpenChange }: EditUserDialogProps) {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [role, setRole] = useState('')
  const [errors, setErrors] = useState<FormErrors>({})

  const updateUser = useUpdateUser()

  useEffect(() => {
    if (user) {
      setName(user.name)
      setEmail(user.email)
      setRole(user.role ? String(user.role) : '')
      setErrors({})
      updateUser.reset()
    }
  }, [user])

  function handleClose(value: boolean) {
    if (!value) {
      setErrors({})
      updateUser.reset()
    }
    onOpenChange(value)
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()

    const validationErrors = validate({ name, email, role })
    setErrors(validationErrors)

    if (Object.keys(validationErrors).length > 0) {
      return
    }

    if (!user) return

    updateUser.mutate(
      {
        id: user.id,
        name,
        email,
        role: Number(role) as Role,
      },
      {
        onSuccess() {
          handleClose(false)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit user</DialogTitle>
          <DialogDescription>Update user account details.</DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="edit-user-name">Name</Label>
            <Input
              id="edit-user-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Full name"
              autoComplete="name"
            />
            {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
          </div>

          <div className="grid gap-2">
            <Label htmlFor="edit-user-email">Email</Label>
            <Input
              id="edit-user-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="user@example.com"
              autoComplete="email"
            />
            {errors.email && <p className="text-sm text-destructive">{errors.email}</p>}
          </div>

          <div className="grid gap-2">
            <Label htmlFor="edit-user-role">Role</Label>
            <Select
              id="edit-user-role"
              value={role}
              onChange={(e) => setRole(e.target.value)}
            >
              <SelectOption value="">Select a role</SelectOption>
              <SelectOption value={String(Role.ADMIN)}>Admin</SelectOption>
              <SelectOption value={String(Role.MEMBER)}>Member</SelectOption>
            </Select>
            {errors.role && <p className="text-sm text-destructive">{errors.role}</p>}
          </div>

          {updateUser.isError && (
            <p className="text-sm text-destructive">
              Failed to update user. Please try again.
            </p>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleClose(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={updateUser.isPending}>
              {updateUser.isPending ? 'Saving…' : 'Save changes'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
