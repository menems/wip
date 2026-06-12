# logout
> client-side logout: clear token from localStorage and redirect to login

**Created**: 2026-03-12 | **Branch**: feat/logout

## Steps
1. [frontend] `useLogout` hook + logout button in nav
   → `useLogout` calls `setToken(null)` and navigates to `/login`; logout button renders in `_authenticated.tsx` nav; clicking it lands the user on the login page
