import { useState, type FormEvent } from "react";
import { useRoles } from "@/features/roles/useRoles";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface UserFormValues {
  name: string;
  email: string;
  /** Only present and required in create mode. */
  password: string;
  roleId: string;
}

interface FormErrors {
  name?: string;
  email?: string;
  password?: string;
  roleId?: string;
}

interface UserFormProps {
  mode: "create" | "edit";
  defaultValues?: Partial<UserFormValues>;
  isSubmitting: boolean;
  /** An error string from the API to display below the form. */
  apiError: string | null;
  onSubmit: (values: UserFormValues) => void;
  onCancel: () => void;
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

function validate(values: UserFormValues, mode: "create" | "edit"): FormErrors {
  const errors: FormErrors = {};

  if (!values.name.trim()) errors.name = "Name is required.";
  if (!values.email.trim()) errors.email = "Email is required.";
  if (!values.roleId) errors.roleId = "Please select a role.";

  if (mode === "create") {
    if (!values.password) {
      errors.password = "Password is required.";
    } else if (values.password.length < 8) {
      errors.password = "Password must be at least 8 characters.";
    }
  }

  return errors;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

/**
 * UserForm is shared between the create and edit pages.
 * In create mode it includes a password field; in edit mode it does not
 * (password changes go through the separate reset-password flow).
 */
export function UserForm({
  mode,
  defaultValues,
  isSubmitting,
  apiError,
  onSubmit,
  onCancel,
}: UserFormProps) {
  const [values, setValues] = useState<UserFormValues>({
    name: defaultValues?.name ?? "",
    email: defaultValues?.email ?? "",
    password: "",
    roleId: defaultValues?.roleId ?? "",
  });
  const [errors, setErrors] = useState<FormErrors>({});

  const { data: roles = [], isLoading: rolesLoading } = useRoles();

  function set(field: keyof UserFormValues, value: string) {
    setValues((prev) => ({ ...prev, [field]: value }));
    // Clear the field error on change
    setErrors((prev) => ({ ...prev, [field]: undefined }));
  }

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fieldErrors = validate(values, mode);
    if (Object.keys(fieldErrors).length > 0) {
      setErrors(fieldErrors);
      return;
    }
    onSubmit(values);
  }

  return (
    <form onSubmit={handleSubmit} noValidate className="space-y-4 max-w-md">
      {/* Name */}
      <div className="space-y-1.5">
        <Label htmlFor="name">Name</Label>
        <Input
          id="name"
          value={values.name}
          onChange={(e) => set("name", e.target.value)}
          disabled={isSubmitting}
          placeholder="Jane Smith"
        />
        {errors.name && (
          <p className="text-xs text-destructive">{errors.name}</p>
        )}
      </div>

      {/* Email */}
      <div className="space-y-1.5">
        <Label htmlFor="email">Email</Label>
        <Input
          id="email"
          type="email"
          value={values.email}
          onChange={(e) => set("email", e.target.value)}
          disabled={isSubmitting}
          placeholder="jane@example.com"
        />
        {errors.email && (
          <p className="text-xs text-destructive">{errors.email}</p>
        )}
      </div>

      {/* Password — create mode only */}
      {mode === "create" && (
        <div className="space-y-1.5">
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            type="password"
            value={values.password}
            onChange={(e) => set("password", e.target.value)}
            disabled={isSubmitting}
            placeholder="Min. 8 characters"
            autoComplete="new-password"
          />
          {errors.password && (
            <p className="text-xs text-destructive">{errors.password}</p>
          )}
        </div>
      )}

      {/* Role */}
      <div className="space-y-1.5">
        <Label htmlFor="role">Role</Label>
        <Select
          value={values.roleId}
          onValueChange={(v) => set("roleId", v)}
          disabled={isSubmitting || rolesLoading}
        >
          <SelectTrigger id="role">
            <SelectValue placeholder="Select a role…" />
          </SelectTrigger>
          <SelectContent>
            {roles.map((role) => (
              <SelectItem key={role.id} value={role.id}>
                {role.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {errors.roleId && (
          <p className="text-xs text-destructive">{errors.roleId}</p>
        )}
      </div>

      {/* API-level error */}
      {apiError && (
        <p role="alert" className="text-sm text-destructive">
          {apiError}
        </p>
      )}

      {/* Actions */}
      <div className="flex gap-2 pt-2">
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting
            ? mode === "create"
              ? "Creating…"
              : "Saving…"
            : mode === "create"
              ? "Create User"
              : "Save Changes"}
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          disabled={isSubmitting}
        >
          Cancel
        </Button>
      </div>
    </form>
  );
}
