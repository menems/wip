import { useNavigate } from "react-router";
import { ArrowLeft } from "lucide-react";
import { useCreateUser } from "@/features/users/useUsers";
import { UserForm, type UserFormValues } from "@/features/users/UserForm";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api";

/**
 * UserCreatePage — form for creating a new user.
 * On success navigates back to /users.
 */
export function UserCreatePage() {
  const navigate = useNavigate();
  const createUser = useCreateUser();

  async function handleSubmit(values: UserFormValues) {
    await createUser.mutateAsync({
      name: values.name,
      email: values.email,
      password: values.password,
      role_id: values.roleId,
    });
    void navigate("/users");
  }

  function getApiError(): string | null {
    if (!createUser.error) return null;
    if (createUser.error instanceof ApiError) return createUser.error.message;
    return "An unexpected error occurred.";
  }

  return (
    <div className="p-6 max-w-2xl space-y-6">
      {/* Back link */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => void navigate("/users")}
        className="-ml-2"
      >
        <ArrowLeft className="mr-2 h-4 w-4" aria-hidden />
        Back to Users
      </Button>

      <div>
        <h1 className="text-2xl font-semibold">New User</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Create a new user account and assign a role.
        </p>
      </div>

      <UserForm
        mode="create"
        isSubmitting={createUser.isPending}
        apiError={getApiError()}
        onSubmit={(v) => void handleSubmit(v)}
        onCancel={() => void navigate("/users")}
      />
    </div>
  );
}
