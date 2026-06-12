import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { ArrowLeft } from "lucide-react";
import { useAuth } from "@/features/auth/useAuth";
import {
  useUser,
  useUpdateUser,
  useDeactivateUser,
  useReactivateUser,
  useResetPassword,
} from "@/features/users/useUsers";
import { UserForm, type UserFormValues } from "@/features/users/UserForm";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";
import { ApiError } from "@/lib/api";

function getApiError(error: unknown): string | null {
  if (!error) return null;
  if (error instanceof ApiError) return error.message;
  return "An unexpected error occurred.";
}

/**
 * UserEditPage allows editing a user's name, email, and role.
 * Also provides deactivate/reactivate and password reset actions
 * gated by the appropriate permissions.
 */
export function UserEditPage() {
  const { id = "" } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { hasPermission } = useAuth();

  const { data: user, isLoading } = useUser(id);
  const updateUser = useUpdateUser(id);
  const deactivate = useDeactivateUser();
  const reactivate = useReactivateUser();
  const resetPassword = useResetPassword();

  const [newPassword, setNewPassword] = useState("");
  const [pwDialogOpen, setPwDialogOpen] = useState(false);
  const [pwError, setPwError] = useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="flex justify-center p-12">
        <Spinner label="Loading user…" />
      </div>
    );
  }

  if (!user) {
    return (
      <div className="p-6">
        <p className="text-destructive">User not found.</p>
      </div>
    );
  }

  async function handleUpdate(values: UserFormValues) {
    await updateUser.mutateAsync({
      name: values.name,
      email: values.email,
      role_id: values.roleId,
    });
    void navigate("/users");
  }

  async function handleDeactivate() {
    await deactivate.mutateAsync(id);
  }

  async function handleReactivate() {
    await reactivate.mutateAsync(id);
  }

  async function handleResetPassword() {
    setPwError(null);
    if (newPassword.length < 8) {
      setPwError("Password must be at least 8 characters.");
      return;
    }
    try {
      await resetPassword.mutateAsync({ id, password: newPassword });
      setNewPassword("");
      setPwDialogOpen(false);
    } catch (err) {
      setPwError(getApiError(err) ?? "Failed to reset password.");
    }
  }

  const canWrite = hasPermission("users", "write");
  const canDelete = hasPermission("users", "delete");

  return (
    <div className="p-6 max-w-2xl space-y-8">
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

      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">{user.name}</h1>
          <p className="text-sm text-muted-foreground mt-1">{user.email}</p>
        </div>
        <Badge variant={user.is_active ? "default" : "outline"}>
          {user.is_active ? "Active" : "Inactive"}
        </Badge>
      </div>

      {/* Edit form */}
      {canWrite ? (
        <section className="space-y-4">
          <h2 className="text-lg font-medium">Profile</h2>
          <UserForm
            mode="edit"
            defaultValues={{
              name: user.name,
              email: user.email,
              roleId: user.roles[0]?.id ?? "",
            }}
            isSubmitting={updateUser.isPending}
            apiError={getApiError(updateUser.error)}
            onSubmit={(v) => void handleUpdate(v)}
            onCancel={() => void navigate("/users")}
          />
        </section>
      ) : (
        /* Read-only view */
        <section className="space-y-2 text-sm">
          <h2 className="text-lg font-medium">Profile</h2>
          <p>
            <span className="font-medium">Name:</span> {user.name}
          </p>
          <p>
            <span className="font-medium">Email:</span> {user.email}
          </p>
          <p className="flex items-center gap-1">
            <span className="font-medium">Roles:</span>
            {user.roles.map((r) => (
              <Badge key={r.id} variant="secondary">
                {r.name}
              </Badge>
            ))}
          </p>
        </section>
      )}

      {/* Deactivate / Reactivate */}
      {canDelete && (
        <section className="space-y-3 border-t pt-6">
          <h2 className="text-lg font-medium">Account status</h2>
          {getApiError(deactivate.error) && (
            <p role="alert" className="text-sm text-destructive">
              {getApiError(deactivate.error)}
            </p>
          )}
          {user.is_active ? (
            <div className="flex items-center gap-4">
              <p className="text-sm text-muted-foreground">
                Deactivating blocks the user from logging in. The account and
                its data are preserved.
              </p>
              <Button
                variant="destructive"
                size="sm"
                disabled={deactivate.isPending}
                onClick={() => void handleDeactivate()}
              >
                {deactivate.isPending ? "Deactivating…" : "Deactivate"}
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-4">
              <p className="text-sm text-muted-foreground">
                This account is currently inactive. Reactivating restores
                login access.
              </p>
              <Button
                variant="outline"
                size="sm"
                disabled={reactivate.isPending}
                onClick={() => void handleReactivate()}
              >
                {reactivate.isPending ? "Reactivating…" : "Reactivate"}
              </Button>
            </div>
          )}
        </section>
      )}

      {/* Reset password */}
      {canWrite && (
        <section className="space-y-3 border-t pt-6">
          <h2 className="text-lg font-medium">Password</h2>
          <p className="text-sm text-muted-foreground">
            Set a new password for this user. They will need to use it on their
            next login.
          </p>
          <Dialog open={pwDialogOpen} onOpenChange={setPwDialogOpen}>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">
                Reset Password
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Reset password</DialogTitle>
                <DialogDescription>
                  Enter a new password for <strong>{user.name}</strong>. Min.
                  8 characters.
                </DialogDescription>
              </DialogHeader>

              <div className="space-y-1.5 py-2">
                <Label htmlFor="new-password">New password</Label>
                <Input
                  id="new-password"
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  disabled={resetPassword.isPending}
                  autoComplete="new-password"
                />
                {pwError && (
                  <p role="alert" className="text-xs text-destructive">
                    {pwError}
                  </p>
                )}
              </div>

              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setPwDialogOpen(false)}
                  disabled={resetPassword.isPending}
                >
                  Cancel
                </Button>
                <Button
                  onClick={() => void handleResetPassword()}
                  disabled={resetPassword.isPending}
                >
                  {resetPassword.isPending ? "Saving…" : "Save Password"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </section>
      )}
    </div>
  );
}
