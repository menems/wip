-- =============================================================================
-- Migration 002: Seed data — admin role + admin user
-- Idempotent: uses INSERT ... ON CONFLICT DO NOTHING
-- =============================================================================

-- ---------------------------------------------------------------------------
-- Admin role (system role — cannot be deleted)
-- ---------------------------------------------------------------------------
INSERT INTO roles (id, name, description, is_system)
VALUES (
  'a0000000-0000-0000-0000-000000000001',
  'admin',
  'Full system access. Cannot be deleted.',
  true
)
ON CONFLICT (name) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Admin role permissions — all valid (resource, action) combinations
-- ---------------------------------------------------------------------------
INSERT INTO role_permissions (role_id, resource, action)
VALUES
  -- users
  ('a0000000-0000-0000-0000-000000000001', 'users',      'read'),
  ('a0000000-0000-0000-0000-000000000001', 'users',      'write'),
  ('a0000000-0000-0000-0000-000000000001', 'users',      'delete'),
  -- roles
  ('a0000000-0000-0000-0000-000000000001', 'roles',      'read'),
  ('a0000000-0000-0000-0000-000000000001', 'roles',      'write'),
  ('a0000000-0000-0000-0000-000000000001', 'roles',      'delete'),
  -- audit_logs
  ('a0000000-0000-0000-0000-000000000001', 'audit_logs', 'read')
ON CONFLICT (role_id, resource, action) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Default admin user
-- Password: "changeme" hashed with bcrypt cost 12
-- IMPORTANT: Change this password immediately after first login.
-- Hash generated with: bcrypt(cost=12, password="changeme")
-- ---------------------------------------------------------------------------
INSERT INTO users (id, email, name, password_hash, is_active)
VALUES (
  'b0000000-0000-0000-0000-000000000001',
  'admin@example.com',
  'Admin',
  '$2a$12$x28r3UtDQFbjBZloVudxCOOwGl4EBizsICHUqKGlsBJHn5whEUxTy',
  true
)
ON CONFLICT (email) DO NOTHING;

-- Assign admin role to default admin user
INSERT INTO user_roles (user_id, role_id)
VALUES (
  'b0000000-0000-0000-0000-000000000001',
  'a0000000-0000-0000-0000-000000000001'
)
ON CONFLICT DO NOTHING;
