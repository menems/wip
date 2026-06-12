import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { ArrowLeft } from "lucide-react";
import { useAuth } from "@/features/auth/useAuth";
import {
  useRole,
  useUpdateRole,
  useDeleteRole,
} from "@/features/roles/useRoles";
import {
  RoleForm,
  type RoleFormValues,
  permissionsSetToArray,
  permissionsArrayToSet,
} from "@/features/roles/RoleForm";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ApiError } from "@/lib/api";

function getApiError(error: unknown): string | null {
  if (!error) return null;
  if (error instanceof ApiError) return error.message;
  return "An unexpected error occurred.";
}

/**
 * RoleEditPage allows editing a role's name, description, and permission set.
 * System roles are displayed read-only — the form disables all inputs.
 * Non-system roles can also be deleted (with a confirmation dialog), provided
 * they are not assigned to any users.
 */
export function RoleEditPage() {
  const { id = "" } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { hasPermission } = useAuth();

  const { data: role, isLoading } = useRole(id);
  const updateRole = useUpdateRole(id);
  const deleteRole = useDeleteRole();

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  if (isLoading) {
    return (
      <div className="flex justify-center p-12">
        <Spinner label="Loading role…" />
      </div>
    );
  }

  if (!role) {
    return (
      <div className="p-6">
        <p className="text-destructive">Role not found.</p>
      </div>
    );
  }

  async function handleUpdate(values: RoleFormValues) {
    await updateRole.mutateAsync({
      name: values.name,
      description: values.description || undefined,
      permissions: permissionsSetToArray(values.permissions),
    });
    void navigate("/roles");
  }

  async function handleDelete() {
    await deleteRole.mutateAsync(id);
    void navigate("/roles");
  }

  const canWrite = hasPermission("roles", "write");
  const canDelete = hasPermission("roles", "delete");

  return (
    <div className="p-6 max-w-2xl space-y-8">
      {/* Back link */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => void navigate("/roles")}
        className="-ml-2"
      >
        <ArrowLeft className="mr-2 h-4 w-4" aria-hidden />
        Back to Roles
      </Button>

      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">{role.name}</h1>
          {role.description && (
            <p className="text-sm text-muted-foreground mt-1">
              {role.description}
            </p>
          )}
        </div>
        {role.is_system && (
          <Badge variant="secondary">System</Badge>
        )}
      </div>

      {/* Edit form */}
      <section className="space-y-4">
        <h2 className="text-lg font-medium">
          {role.is_system ? "Permissions (read-only)" : "Details & Permissions"}
        </h2>
        <RoleForm
          mode="edit"
          defaultValues={{
            name: role.name,
            description: role.description ?? "",
            permissions: permissionsArrayToSet(role.permissions),
          }}
          isSubmitting={updateRole.isPending}
          isSystemRole={role.is_system}
          apiError={getApiError(updateRole.error)}
          onSubmit={(v) => void handleUpdate(v)}
          onCancel={() => void navigate("/roles")}
        />
      </section>

      {/* Delete section — only for non-system roles with delete permission */}
      {canDelete && !role.is_system && (
        <section className="space-y-3 border-t pt-6">
          <h2 className="text-lg font-medium text-destructive">Danger zone</h2>
          <p className="text-sm text-muted-foreground">
            Deleting a role removes it permanently. This action will fail if
            the role is currently assigned to any users.
          </p>

          <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
            <DialogTrigger asChild>
              <Button variant="destructive" size="sm">
                Delete Role
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Delete role</DialogTitle>
                <DialogDescription>
                  Are you sure you want to delete{" "}
                  <strong>{role.name}</strong>? This cannot be undone.
                </DialogDescription>
              </DialogHeader>

              {getApiError(deleteRole.error) && (
                <p role="alert" className="text-sm text-destructive">
                  {getApiError(deleteRole.error)}
                </p>
              )}

              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setDeleteDialogOpen(false)}
                  disabled={deleteRole.isPending}
                >
                  Cancel
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => void handleDelete()}
                  disabled={deleteRole.isPending}
                >
                  {deleteRole.isPending ? "Deleting…" : "Delete"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </section>
      )}

      {/* Informational note if user only has read access */}
      {!canWrite && !role.is_system && (
        <p className="text-sm text-muted-foreground italic">
          You have read-only access to roles.
        </p>
      )}
    </div>
  );
}
