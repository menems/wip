/**
 * useAuth — convenience re-export from AuthContext.
 *
 * Import from here rather than directly from AuthContext.tsx to keep
 * the import path short in feature modules:
 *   import { useAuth } from "@/features/auth/useAuth";
 */
export { useAuth } from "./AuthContext";
