-- Seed a default admin user for local development.
-- Credentials: admin@localhost / admin1234
-- To regenerate the hash: go run ./cmd/gen-bcrypt-hash <password>
INSERT INTO users (id, email, name, password_hash, role)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@localhost',
    'Admin',
    '$2a$10$N2lpxVEnve9Jv2RY0yZcjuVKxtwr469Sh0ZWEmUxm9ZqHKiGGliYO',
    'admin'
) ON CONFLICT (email) DO NOTHING;
