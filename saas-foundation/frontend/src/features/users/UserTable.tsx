import { Link } from "react-router";
import { Pencil, UserX, UserCheck } from "lucide-react";
import type { UserItem } from "@/lib/api";
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

interface UserTableProps {
  users: UserItem[];
  isLoading: boolean;
  /** Whether to show the Edit action column (requires users:write). */
  canEdit: boolean;
  /** Whether to show the Deactivate action (requires users:delete). */
  canDeactivate: boolean;
  onDeactivate: (id: string) => void;
  onReactivate: (id: string) => void;
}

/**
 * UserTable renders a list of users in a table.
 * It is a pure presentational component — all data and callbacks come from props.
 * The full DataTable with sorting and filtering will be added in task 2.4.
 */
export function UserTable({
  users,
  isLoading,
  canEdit,
  canDeactivate,
  onDeactivate,
  onReactivate,
}: UserTableProps) {
  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner label="Loading users…" />
      </div>
    );
  }

  if (users.length === 0) {
    return (
      <div className="flex justify-center py-12 text-sm text-muted-foreground">
        No users found.
      </div>
    );
  }

  const showActions = canEdit || canDeactivate;

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Roles</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Created</TableHead>
          {showActions && <TableHead className="text-right">Actions</TableHead>}
        </TableRow>
      </TableHeader>
      <TableBody>
        {users.map((user) => (
          <TableRow key={user.id}>
            {/* Name + email */}
            <TableCell>
              <div className="font-medium">{user.name}</div>
              <div className="text-xs text-muted-foreground">{user.email}</div>
            </TableCell>

            {/* Roles */}
            <TableCell>
              <div className="flex flex-wrap gap-1">
                {user.roles.length === 0 ? (
                  <span className="text-xs text-muted-foreground">—</span>
                ) : (
                  user.roles.map((role) => (
                    <Badge key={role.id} variant="secondary">
                      {role.name}
                    </Badge>
                  ))
                )}
              </div>
            </TableCell>

            {/* Status */}
            <TableCell>
              <Badge variant={user.is_active ? "default" : "outline"}>
                {user.is_active ? "Active" : "Inactive"}
              </Badge>
            </TableCell>

            {/* Created at */}
            <TableCell className="text-sm text-muted-foreground">
              {new Date(user.created_at).toLocaleDateString()}
            </TableCell>

            {/* Actions */}
            {showActions && (
              <TableCell className="text-right">
                <div className="flex items-center justify-end gap-2">
                  {canEdit && (
                    <Button variant="ghost" size="icon" asChild>
                      <Link to={`/users/${user.id}`} aria-label={`Edit ${user.name}`}>
                        <Pencil className="h-4 w-4" aria-hidden />
                      </Link>
                    </Button>
                  )}

                  {canDeactivate && (
                    user.is_active ? (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onDeactivate(user.id)}
                        aria-label={`Deactivate ${user.name}`}
                      >
                        <UserX className="h-4 w-4 text-destructive" aria-hidden />
                      </Button>
                    ) : (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onReactivate(user.id)}
                        aria-label={`Reactivate ${user.name}`}
                      >
                        <UserCheck className="h-4 w-4 text-green-600" aria-hidden />
                      </Button>
                    )
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
