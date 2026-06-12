import { useNavigate } from "react-router";
import { ArrowLeft } from "lucide-react";
import { useCreateRole } from "@/features/roles/useRoles";
import {
  RoleForm,
  type RoleFormValues,
  permissionsSetToArray,
} from "@/features/roles/RoleForm";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api";

/**
 * RoleCreatePage — form for creating a new role with a permission matrix.
 * On success navigates back to /roles.
 */
export function RoleCreatePage() {
  const navigate = useNavigate();
  const createRole = useCreateRole();

  async function handleSubmit(values: RoleFormValues) {
    await createRole.mutateAsync({
      name: values.name,
      description: values.description || undefined,
      permissions: permissionsSetToArray(values.permissions),
    });
    void navigate("/roles");
  }

  function getApiError(): string | null {
    if (!createRole.error) return null;
    if (createRole.error instanceof ApiError) return createRole.error.message;
    return "An unexpected error occurred.";
  }

  return (
    <div className="p-6 max-w-2xl space-y-6">
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

      <div>
        <h1 className="text-2xl font-semibold">New Role</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Create a new role and define its permission set.
        </p>
      </div>

      <RoleForm
        mode="create"
        isSubmitting={createRole.isPending}
        apiError={getApiError()}
        onSubmit={(v) => void handleSubmit(v)}
        onCancel={() => void navigate("/roles")}
      />
    </div>
  );
}
