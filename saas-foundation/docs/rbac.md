# RBAC Permission Table

| Resource     | Actions                   | Notes                                              |
|--------------|---------------------------|----------------------------------------------------|
| `users`      | `read`, `write`, `delete` | `delete` = deactivate/reactivate (soft delete)     |
| `roles`      | `read`, `write`, `delete` | `delete` blocked for system roles or assigned ones |
| `audit_logs` | `read`                    | Read-only; no write/delete at application layer    |
