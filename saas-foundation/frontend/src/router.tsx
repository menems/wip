import { createBrowserRouter, Navigate } from "react-router";
import { ProtectedRoute } from "@/components/ProtectedRoute";
import { AppLayout } from "@/components/Navigation/AppLayout";
import { LoginPage } from "@/pages/LoginPage";
import { DashboardPage } from "@/pages/DashboardPage";
import { NotFoundPage } from "@/pages/NotFoundPage";
import { UsersListPage } from "@/pages/UsersListPage";
import { UserCreatePage } from "@/pages/UserCreatePage";
import { UserEditPage } from "@/pages/UserEditPage";
import { RolesListPage } from "@/pages/RolesListPage";
import { RoleCreatePage } from "@/pages/RoleCreatePage";
import { RoleEditPage } from "@/pages/RoleEditPage";

/**
 * Application router.
 *
 * Authenticated routes are nested under the AppLayout layout route so they
 * all share the Sidebar + TopBar shell. ProtectedRoute at the layout level
 * handles authentication; child routes add a second ProtectedRoute only when
 * a specific permission is required.
 *
 * Feature pages (users, roles, audit-logs) are filled in tasks 1.11, 1.12, 2.3.
 */
export const router = createBrowserRouter([
  // -------------------------------------------------------------------------
  // Public routes (no shell)
  // -------------------------------------------------------------------------
  {
    path: "/login",
    element: <LoginPage />,
  },

  // -------------------------------------------------------------------------
  // Authenticated routes — all wrapped in AppLayout
  // -------------------------------------------------------------------------
  {
    path: "/",
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      // Redirect root → /dashboard
      { index: true, element: <Navigate to="/dashboard" replace /> },

      // Dashboard — any authenticated user
      { path: "dashboard", element: <DashboardPage /> },

      // Users — requires users:read (write/delete checked per action in the pages)
      {
        path: "users",
        element: (
          <ProtectedRoute permission="users:read">
            <UsersListPage />
          </ProtectedRoute>
        ),
      },
      {
        path: "users/new",
        element: (
          <ProtectedRoute permission="users:write">
            <UserCreatePage />
          </ProtectedRoute>
        ),
      },
      {
        path: "users/:id",
        element: (
          <ProtectedRoute permission="users:read">
            <UserEditPage />
          </ProtectedRoute>
        ),
      },

      // Roles — requires roles:read (write/delete checked per action in the pages)
      {
        path: "roles",
        element: (
          <ProtectedRoute permission="roles:read">
            <RolesListPage />
          </ProtectedRoute>
        ),
      },
      {
        path: "roles/new",
        element: (
          <ProtectedRoute permission="roles:write">
            <RoleCreatePage />
          </ProtectedRoute>
        ),
      },
      {
        path: "roles/:id",
        element: (
          <ProtectedRoute permission="roles:read">
            <RoleEditPage />
          </ProtectedRoute>
        ),
      },

      // Audit logs — requires audit_logs:read
      {
        path: "audit-logs",
        element: (
          <ProtectedRoute permission="audit_logs:read">
            <div className="p-6">Audit Logs — coming in task 2.3</div>
          </ProtectedRoute>
        ),
      },
    ],
  },

  // -------------------------------------------------------------------------
  // 404 fallback
  // -------------------------------------------------------------------------
  { path: "*", element: <NotFoundPage /> },
]);
