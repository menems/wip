import { useState, type FormEvent } from 'react'
import { Role } from '../gen/user_pb'
import { useCreateUser } from '../hooks/useUsers'
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

interface CreateUserDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface FormErrors {
  name?: string
  email?: string
  password?: string
  role?: string
}

function validate(fields: { name: string; email: string; password: string; role: string }): FormErrors {
  const errors: FormErrors = {}

  if (!fields.name.trim()) {
    errors.name = 'Name is required'
  }

  if (!fields.email.trim()) {
    errors.email = 'Email is required'
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(fields.email)) {
    errors.email = 'Invalid email address'
  }

  if (!fields.password) {
    errors.password = 'Password is required'
  } else if (fields.password.length < 8) {
    errors.password = 'Password must be at least 8 characters'
  }

  if (!fields.role) {
    errors.role = 'Role is required'
  }

  return errors
}

export function CreateUserDialog({ open, onOpenChange }: CreateUserDialogProps) {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('')
  const [errors, setErrors] = useState<FormErrors>({})

  const createUser = useCreateUser()

  function resetForm() {
    setName('')
    setEmail('')
    setPassword('')
    setRole('')
    setErrors({})
    createUser.reset()
  }

  function handleClose(value: boolean) {
    if (!value) {
      resetForm()
    }
    onOpenChange(value)
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()

    const validationErrors = validate({ name, email, password, role })
    setErrors(validationErrors)

    if (Object.keys(validationErrors).length > 0) {
      return
    }

    createUser.mutate(
      {
        name,
        email,
        password,
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
          <DialogTitle>Add user</DialogTitle>
          <DialogDescription>Create a new user account.</DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="create-user-name">Name</Label>
            <Input
              id="create-user-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Full name"
              autoComplete="name"
            />
            {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
          </div>

          <div className="grid gap-2">
            <Label htmlFor="create-user-email">Email</Label>
            <Input
              id="create-user-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="user@example.com"
              autoComplete="email"
            />
            {errors.email && <p className="text-sm text-destructive">{errors.email}</p>}
          </div>

          <div className="grid gap-2">
            <Label htmlFor="create-user-password">Password</Label>
            <Input
              id="create-user-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Minimum 8 characters"
              autoComplete="new-password"
            />
            {errors.password && <p className="text-sm text-destructive">{errors.password}</p>}
          </div>

          <div className="grid gap-2">
            <Label htmlFor="create-user-role">Role</Label>
            <Select
              id="create-user-role"
              value={role}
              onChange={(e) => setRole(e.target.value)}
            >
              <SelectOption value="">Select a role</SelectOption>
              <SelectOption value={String(Role.ADMIN)}>Admin</SelectOption>
              <SelectOption value={String(Role.MEMBER)}>Member</SelectOption>
            </Select>
            {errors.role && <p className="text-sm text-destructive">{errors.role}</p>}
          </div>

          {createUser.isError && (
            <p className="text-sm text-destructive">
              Failed to create user. Please try again.
            </p>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleClose(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={createUser.isPending}>
              {createUser.isPending ? 'Creating…' : 'Create user'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
