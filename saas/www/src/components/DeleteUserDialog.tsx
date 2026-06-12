import type { User } from '../gen/user_pb'
import { useDeleteUser } from '../hooks/useUsers'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from './ui/dialog'
import { Button } from './ui/button'

interface DeleteUserDialogProps {
  user: User | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DeleteUserDialog({ user, open, onOpenChange }: DeleteUserDialogProps) {
  const deleteUser = useDeleteUser()

  function handleClose(value: boolean) {
    if (!value) {
      deleteUser.reset()
    }
    onOpenChange(value)
  }

  function handleConfirm() {
    if (!user) return

    deleteUser.mutate(
      { id: user.id },
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
          <DialogTitle>Delete user</DialogTitle>
          <DialogDescription>
            Are you sure you want to delete <strong>{user?.name}</strong>? This action cannot be
            undone.
          </DialogDescription>
        </DialogHeader>

        {deleteUser.isError && (
          <p className="text-sm text-destructive">
            Failed to delete user. Please try again.
          </p>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => handleClose(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleConfirm}
            disabled={deleteUser.isPending}
          >
            {deleteUser.isPending ? 'Deleting…' : 'Delete user'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
