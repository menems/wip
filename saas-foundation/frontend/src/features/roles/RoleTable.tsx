import { Link } from "react-router";
import { Pencil, Trash2 } from "lucide-react";
import type { RoleItem } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Spinner } from "@/components/ui/spinner";

interface RoleTableProps {
  roles: RoleItem[];
  isLoading: boolean;
  /** Whether to show the Edit action (requires roles:write). */
  canEdit: boolean;
  /** Whether to show the Delete action (requires roles:delete). */
  canDelete: boolean;
  onDelete: (role: RoleItem) => void;
}

/**
 * RoleTable renders a list of roles in a table.
 * It is a pure presentational component — all data and callbacks come from props.
 * System roles display a "System" badge and their Delete button is omitted since
 * the backend blocks deletion of system roles.
 */
export function RoleTable({
  roles,
  isLoading,
  canEdit,
  canDelete,
  onDelete,
}: RoleTableProps) {
  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner label="Loading roles…" />
      </div>
    );
  }

  if (roles.length === 0) {
    return (
      <div className="flex justify-center py-12 text-sm text-muted-foreground">
        No roles found.
      </div>
    );
  }

  const showActions = canEdit || canDelete;

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Description</TableHead>
          <TableHead>Permissions</TableHead>
          {showActions && (
            <TableHead className="text-right">Actions</TableHead>
          )}
        </TableRow>
      </TableHeader>
      <TableBody>
        {roles.map((role) => (
          <TableRow key={role.id}>
            {/* Name + system badge */}
            <TableCell>
              <div className="flex items-center gap-2">
                <span className="font-medium">{role.name}</span>
                {role.is_system && (
                  <Badge variant="secondary" className="text-xs">
                    System
                  </Badge>
                )}
              </div>
            </TableCell>

            {/* Description */}
            <TableCell className="text-sm text-muted-foreground">
              {role.description ?? <span className="italic">—</span>}
            </TableCell>

            {/* Permission count */}
            <TableCell className="text-sm text-muted-foreground">
              {role.permissions.length === 0 ? (
                <span className="italic">None</span>
              ) : (
                `${role.permissions.length} permission${role.permissions.length !== 1 ? "s" : ""}`
              )}
            </TableCell>

            {/* Actions */}
            {showActions && (
              <TableCell className="text-right">
                <div className="flex items-center justify-end gap-2">
                  {canEdit && (
                    <Button variant="ghost" size="icon" asChild>
                      <Link
                        to={`/roles/${role.id}`}
                        aria-label={`Edit ${role.name}`}
                      >
                        <Pencil className="h-4 w-4" aria-hidden />
                      </Link>
                    </Button>
                  )}

                  {/* System roles cannot be deleted */}
                  {canDelete && !role.is_system && (
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onDelete(role)}
                      aria-label={`Delete ${role.name}`}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" aria-hidden />
                    </Button>
                  )}
                </div>
              </TableCell>
            )}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
