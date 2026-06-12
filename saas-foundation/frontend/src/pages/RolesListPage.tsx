import { useState } from "react";
import { useNavigate } from "react-router";
import { PlusCircle } from "lucide-react";
import { useAuth } from "@/features/auth/useAuth";
import { useRoles, useDeleteRole } from "@/features/roles/useRoles";
import { RoleTable } from "@/features/roles/RoleTable";
import type { RoleItem } from "@/lib/api";
import { ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

/**
 * RolesListPage displays all roles with their permission counts.
 * Users with roles:write can create and edit roles.
 * Users with roles:delete can delete non-system roles (with confirmation).
 */
export function RolesListPage() {
  const { hasPermission } = useAuth();
  const navigate = useNavigate();

  const { data: roles = [], isLoading } = useRoles();
  const deleteRole = useDeleteRole();

  const [roleToDelete, setRoleToDelete] = useState<RoleItem | null>(null);

  function getDeleteError(): string | null {
    if (!deleteRole.error) return null;
    if (deleteRole.error instanceof ApiError) return deleteRole.error.message;
    return "An unexpected error occurred.";
  }

  async function handleConfirmDelete() {
    if (!roleToDelete) return;
    try {
      await deleteRole.mutateAsync(roleToDelete.id);
      setRoleToDelete(null);
    } catch {
      // Error displayed in the dialog via getDeleteError()
    }
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Roles</h1>
          {roles.length > 0 && (
            <p className="text-sm text-muted-foreground mt-1">
              {roles.length} role{roles.length !== 1 ? "s" : ""}
            </p>
          )}
        </div>
        {hasPermission("roles", "write") && (
          <Button onClick={() => void navigate("/roles/new")}>
            <PlusCircle className="mr-2 h-4 w-4" aria-hidden />
            New Role
          </Button>
        )}
      </div>

      {/* Table */}
      <RoleTable
        roles={roles}
        isLoading={isLoading}
        canEdit={hasPermission("roles", "write")}
        canDelete={hasPermission("roles", "delete")}
        onDelete={(role) => {
          deleteRole.reset();
          setRoleToDelete(role);
        }}
      />

      {/* Delete confirmation dialog */}
      <Dialog
        open={roleToDelete !== null}
        onOpenChange={(open) => {
          if (!open) setRoleToDelete(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete role</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete{" "}
              <strong>{roleToDelete?.name}</strong>? This cannot be undone.
              The delete will fail if the role is assigned to any users.
            </DialogDescription>
          </DialogHeader>

          {getDeleteError() && (
            <p role="alert" className="text-sm text-destructive">
              {getDeleteError()}
            </p>
          )}

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setRoleToDelete(null)}
              disabled={deleteRole.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => void handleConfirmDelete()}
              disabled={deleteRole.isPending}
            >
              {deleteRole.isPending ? "Deleting…" : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
