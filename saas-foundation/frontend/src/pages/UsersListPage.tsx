import { useState } from "react";
import { useNavigate } from "react-router";
import { PlusCircle } from "lucide-react";
import { useAuth } from "@/features/auth/useAuth";
import {
  useUsers,
  useDeactivateUser,
  useReactivateUser,
} from "@/features/users/useUsers";
import { UserTable } from "@/features/users/UserTable";
import { useDebounce } from "@/hooks/useDebounce";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api";

const PER_PAGE = 25;

/**
 * UsersListPage displays a paginated, searchable list of users with
 * quick deactivate/reactivate actions.
 */
export function UsersListPage() {
  const { hasPermission } = useAuth();
  const navigate = useNavigate();

  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const debouncedSearch = useDebounce(search, 300);

  const { data, isLoading } = useUsers({
    search: debouncedSearch || undefined,
    page,
    per_page: PER_PAGE,
  });

  const deactivate = useDeactivateUser();
  const reactivate = useReactivateUser();

  const users = data?.data ?? [];
  const meta = data?.meta;
  const totalPages = meta ? Math.ceil(meta.total / meta.per_page) : 1;

  function getActionError(error: unknown): string {
    if (error instanceof ApiError) return error.message;
    return "";
  }

  const actionError =
    getActionError(deactivate.error) || getActionError(reactivate.error);

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Users</h1>
          {meta && (
            <p className="text-sm text-muted-foreground mt-1">
              {meta.total} user{meta.total !== 1 ? "s" : ""}
            </p>
          )}
        </div>
        {hasPermission("users", "write") && (
          <Button onClick={() => void navigate("/users/new")}>
            <PlusCircle className="mr-2 h-4 w-4" aria-hidden />
            New User
          </Button>
        )}
      </div>

      {/* Search */}
      <Input
        placeholder="Search by name or email…"
        value={search}
        onChange={(e) => {
          setSearch(e.target.value);
          setPage(1); // reset to first page on new search
        }}
        className="max-w-sm"
        aria-label="Search users"
      />

      {/* Action error (deactivate/reactivate failures) */}
      {actionError && (
        <p role="alert" className="text-sm text-destructive">
          {actionError}
        </p>
      )}

      {/* Table */}
      <UserTable
        users={users}
        isLoading={isLoading}
        canEdit={hasPermission("users", "write")}
        canDeactivate={hasPermission("users", "delete")}
        onDeactivate={(id) => deactivate.mutate(id)}
        onReactivate={(id) => reactivate.mutate(id)}
      />

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center gap-3 text-sm">
          <Button
            variant="outline"
            size="sm"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            Previous
          </Button>
          <span className="text-muted-foreground">
            Page {page} of {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  );
}
