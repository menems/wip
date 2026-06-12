# Plan: user-management-ui
> Frontend UI for CRUD user management (list, create, edit, delete) using existing backend APIs
**Created**: 2026-03-11  |  **Branch**: feat/user-management-ui  |  **Status**: in-progress

## Steps
- [x] `step-01` [frontend] Add shadcn UI components: dialog, table, select, badge
  - **Files**: www/src/components/ui/dialog.tsx (create), www/src/components/ui/table.tsx (create), www/src/components/ui/select.tsx (create), www/src/components/ui/badge.tsx (create)
  - **Commit**: feat(frontend): add dialog, table, select, and badge UI components
  - Install/scaffold shadcn components needed for user management
  - **Done when**: components render without errors, `npm run lint` passes

- [x] `step-02` [frontend] Create user query/mutation hooks
  - **Files**: www/src/hooks/useUsers.ts (create), www/src/hooks/useUsers.test.tsx (create)
  - **Commit**: feat(frontend): add useUsers hooks for CRUD operations
  - **Approval**: required
  - Hooks: `useUsers()`, `useUser(id)`, `useCreateUser()`, `useUpdateUser()`, `useDeleteUser()` wrapping connect-query, following `useAuth.ts` pattern
  - **Done when**: hooks compile, tests pass using MSW mocks

- [x] `step-03` [frontend] Build users list page
  - **Files**: www/src/routes/_authenticated/users.tsx (create)
  - **Commit**: feat(frontend): add users list page with table
  - Authenticated route at `/users` displaying a table of users (name, email, role, created date) using `useUsers()` hook. Includes "Add user" button.
  - **Done when**: route renders table with data from API, accessible at `/users`, `npm run lint` passes

- [x] `step-04` [frontend] Build create-user dialog
  - **Files**: www/src/components/CreateUserDialog.tsx (create)
  - **Commit**: feat(frontend): add create user dialog with form validation
  - Modal dialog with form fields: name, email, password, role (select). Client-side validation. Uses `useCreateUser()` mutation, invalidates user list on success.
  - **Done when**: dialog opens from "Add user" button, creates user, closes and refreshes list on success

- [x] `step-05` [frontend] Build edit-user dialog
  - **Files**: www/src/components/EditUserDialog.tsx (create)
  - **Commit**: feat(frontend): add edit user dialog with pre-filled form
  - Modal dialog pre-filled with existing user data (name, email, role). Uses `useUpdateUser()` mutation.
  - **Done when**: dialog opens from row action, updates user, closes and refreshes list on success

- [x] `step-06` [frontend] Build delete-user confirmation dialog
  - **Files**: www/src/components/DeleteUserDialog.tsx (create)
  - **Commit**: feat(frontend): add delete user confirmation dialog
  - Confirmation dialog showing user name. Uses `useDeleteUser()` mutation.
  - **Done when**: dialog opens from row action, deletes user on confirm, refreshes list

- [x] `step-07` [frontend] Wire dialogs into users page + add nav link
  - **Files**: www/src/routes/_authenticated/users.tsx (modify), www/src/routes/_authenticated.tsx (modify)
  - **Commit**: feat(frontend): wire user CRUD dialogs and add nav to users page
  - Integrate Create/Edit/Delete dialogs into users page with row actions. Add navigation link to `/users` in authenticated layout.
  - **Done when**: full CRUD flow works end-to-end, nav link visible, `npm run lint` passes

## Log
- [2026-03-11] step-01 done by frontend — feat(frontend): add dialog, table, select, and badge UI components
- [2026-03-11] step-02 done by frontend — feat(frontend): add useUsers hooks for CRUD operations
- [2026-03-11] step-03 done by frontend — feat(frontend): add users list page with table
- [2026-03-11] step-04 done by frontend — feat(frontend): add create user dialog with form validation
- [2026-03-11] step-05 done by frontend — feat(frontend): add edit user dialog with pre-filled form
- [2026-03-11] step-06 done by frontend — feat(frontend): add delete user confirmation dialog
- [2026-03-11] step-07 done by frontend — feat(frontend): wire user CRUD dialogs and add nav to users page

## Notes
