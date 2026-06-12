import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";

// ---------------------------------------------------------------------------
// Permission matrix definition
// ---------------------------------------------------------------------------

/**
 * All valid resource/action combinations as defined in the spec.
 * audit_logs only supports "read".
 */
export const PERMISSION_MATRIX = [
  { resource: "users", actions: ["read", "write", "delete"] },
  { resource: "roles", actions: ["read", "write", "delete"] },
  { resource: "audit_logs", actions: ["read"] },
] as const;

/** All possible actions across all resources — used to render column headers. */
export const ALL_ACTIONS = ["read", "write", "delete"] as const;

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface RoleFormValues {
  name: string;
  description: string;
  /** Permission keys in "resource:action" format, e.g. "users:read". */
  permissions: ReadonlySet<string>;
}

interface FormErrors {
  name?: string;
  permissions?: string;
}

interface RoleFormProps {
  mode: "create" | "edit";
  defaultValues?: Partial<RoleFormValues>;
  isSubmitting: boolean;
  /** Whether the role is a system role (disables the permission matrix). */
  isSystemRole?: boolean;
  /** An error string from the API to display below the form. */
  apiError: string | null;
  onSubmit: (values: RoleFormValues) => void;
  onCancel: () => void;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function validate(values: RoleFormValues): FormErrors {
  const errors: FormErrors = {};
  if (!values.name.trim()) errors.name = "Name is required.";
  return errors;
}

/** Convert a Set<"resource:action"> to the API array format. */
export function permissionsSetToArray(
  permissions: ReadonlySet<string>
): Array<{ resource: string; action: string }> {
  return Array.from(permissions).map((key) => {
    const [resource, action] = key.split(":") as [string, string];
    return { resource, action };
  });
}

/** Convert an API permissions array to a Set<"resource:action">. */
export function permissionsArrayToSet(
  permissions: Array<{ resource: string; action: string }>
): Set<string> {
  return new Set(permissions.map(({ resource, action }) => `${resource}:${action}`));
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

/**
 * RoleForm is shared between the create and edit pages.
 *
 * The permission matrix renders a row per resource and a column per action.
 * Cells that have no valid action for a resource (e.g. audit_logs:write) are
 * rendered as disabled to communicate that the combination does not exist.
 *
 * System roles (`isSystemRole=true`) show the matrix read-only to prevent
 * accidental modification of built-in permissions.
 */
export function RoleForm({
  mode,
  defaultValues,
  isSubmitting,
  isSystemRole = false,
  apiError,
  onSubmit,
  onCancel,
}: RoleFormProps) {
  const [name, setName] = useState(defaultValues?.name ?? "");
  const [description, setDescription] = useState(
    defaultValues?.description ?? ""
  );
  const [permissions, setPermissions] = useState<Set<string>>(
    () => new Set(defaultValues?.permissions ?? [])
  );
  const [errors, setErrors] = useState<FormErrors>({});

  function togglePermission(key: string) {
    setPermissions((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  }

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const values: RoleFormValues = { name, description, permissions };
    const fieldErrors = validate(values);
    if (Object.keys(fieldErrors).length > 0) {
      setErrors(fieldErrors);
      return;
    }
    onSubmit(values);
  }

  const disabled = isSubmitting || isSystemRole;

  return (
    <form onSubmit={handleSubmit} noValidate className="space-y-6 max-w-lg">
      {/* Name */}
      <div className="space-y-1.5">
        <Label htmlFor="role-name">Name</Label>
        <Input
          id="role-name"
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setErrors((prev) => ({ ...prev, name: undefined }));
          }}
          disabled={isSubmitting}
          placeholder="e.g. viewer"
        />
        {errors.name && (
          <p className="text-xs text-destructive">{errors.name}</p>
        )}
      </div>

      {/* Description */}
      <div className="space-y-1.5">
        <Label htmlFor="role-description">
          Description{" "}
          <span className="text-muted-foreground font-normal">(optional)</span>
        </Label>
        <Input
          id="role-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          disabled={isSubmitting}
          placeholder="Short description of this role"
        />
      </div>

      <Separator />

      {/* Permission matrix */}
      <div className="space-y-3">
        <div>
          <h3 className="text-sm font-medium">Permissions</h3>
          {isSystemRole && (
            <p className="text-xs text-muted-foreground mt-0.5">
              System role permissions cannot be modified.
            </p>
          )}
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm border-collapse">
            <thead>
              <tr>
                <th className="text-left font-medium py-2 pr-4 w-32">
                  Resource
                </th>
                {ALL_ACTIONS.map((action) => (
                  <th
                    key={action}
                    className="text-center font-medium py-2 px-4 capitalize w-20"
                  >
                    {action}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {PERMISSION_MATRIX.map(({ resource, actions }) => (
                <tr
                  key={resource}
                  className="border-t border-border/50 hover:bg-muted/30"
                >
                  <td className="py-2.5 pr-4 font-mono text-xs text-muted-foreground">
                    {resource}
                  </td>
                  {ALL_ACTIONS.map((action) => {
                    const key = `${resource}:${action}`;
                    const isValid = (actions as readonly string[]).includes(
                      action
                    );
                    return (
                      <td key={action} className="text-center py-2.5 px-4">
                        {isValid ? (
                          <Checkbox
                            id={key}
                            checked={permissions.has(key)}
                            onCheckedChange={() => togglePermission(key)}
                            disabled={disabled}
                            aria-label={`${resource} ${action}`}
                          />
                        ) : (
                          <span
                            className="inline-block w-4 h-4 rounded border border-border/30 bg-muted/20"
                            aria-label={`${resource} ${action} not applicable`}
                          />
                        )}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* API-level error */}
      {apiError && (
        <p role="alert" className="text-sm text-destructive">
          {apiError}
        </p>
      )}

      {/* Actions */}
      <div className="flex gap-2 pt-2">
        {!isSystemRole && (
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting
              ? mode === "create"
                ? "Creating…"
                : "Saving…"
              : mode === "create"
                ? "Create Role"
                : "Save Changes"}
          </Button>
        )}
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          disabled={isSubmitting}
        >
          {isSystemRole ? "Back" : "Cancel"}
        </Button>
      </div>
    </form>
  );
}
