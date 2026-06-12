CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    email       TEXT        UNIQUE NOT NULL,
    password_hash TEXT      NOT NULL,
    name        TEXT        NOT NULL DEFAULT '',
    avatar_url  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
