-- =============================================================================
-- Migration 001: Initial schema
-- Tables: users, roles, user_roles, role_permissions, refresh_tokens, audit_logs
-- =============================================================================

-- ---------------------------------------------------------------------------
-- Utility: updated_at trigger function (reused by all mutable tables)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- users
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  email         VARCHAR(255) NOT NULL,
  name          VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  is_active     BOOLEAN     NOT NULL DEFAULT true,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE TRIGGER users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ---------------------------------------------------------------------------
-- roles
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS roles (
  id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  name        VARCHAR(100) NOT NULL,
  description TEXT,
  is_system   BOOLEAN      NOT NULL DEFAULT false,
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

  CONSTRAINT roles_name_unique UNIQUE (name)
);

CREATE TRIGGER roles_updated_at
  BEFORE UPDATE ON roles
  FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ---------------------------------------------------------------------------
-- user_roles  (junction)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_roles (
  user_id     UUID        NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
  role_id     UUID        NOT NULL REFERENCES roles(id)  ON DELETE CASCADE,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS user_roles_user_id_idx ON user_roles (user_id);
CREATE INDEX IF NOT EXISTS user_roles_role_id_idx ON user_roles (role_id);

-- ---------------------------------------------------------------------------
-- role_permissions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS role_permissions (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  role_id    UUID         NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  resource   VARCHAR(100) NOT NULL,
  action     VARCHAR(50)  NOT NULL,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),

  CONSTRAINT role_permissions_unique UNIQUE (role_id, resource, action)
);

CREATE INDEX IF NOT EXISTS role_permissions_role_id_idx ON role_permissions (role_id);

-- ---------------------------------------------------------------------------
-- refresh_tokens
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS refresh_tokens (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash VARCHAR(255) NOT NULL,
  expires_at TIMESTAMPTZ  NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),

  CONSTRAINT refresh_tokens_hash_unique UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS refresh_tokens_token_hash_idx ON refresh_tokens (token_hash);
CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx    ON refresh_tokens (user_id);

-- ---------------------------------------------------------------------------
-- audit_logs  (append-only — no updated_at)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_id      UUID         NOT NULL REFERENCES users(id),
  action        VARCHAR(100) NOT NULL,
  resource_type VARCHAR(100) NOT NULL,
  resource_id   UUID,
  before_state  JSONB,
  after_state   JSONB,
  ip_address    INET,
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_logs_actor_created_idx    ON audit_logs (actor_id, created_at);
CREATE INDEX IF NOT EXISTS audit_logs_resource_type_id_idx ON audit_logs (resource_type, resource_id);
