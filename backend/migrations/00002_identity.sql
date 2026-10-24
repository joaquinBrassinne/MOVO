-- Identity: staff of the businesses and super admins. Customers have no account.
-- Emails are stored already normalized (lowercase, trimmed) by the application;
-- the check makes the database refuse anything else, so the UNIQUE constraint
-- really means "one account per address".

-- +goose Up
CREATE TABLE users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         text        NOT NULL UNIQUE CHECK (email = lower(btrim(email)) AND email <> ''),
    password_hash text        NOT NULL,
    role          text        NOT NULL CHECK (role IN ('business_admin', 'super_admin')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Only the SHA-256 of the opaque token is stored. A rotated token points at its
-- successor through replaced_by, which is how reuse of a stale token is detected.
CREATE TABLE refresh_tokens (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  bytea       NOT NULL UNIQUE CHECK (length(token_hash) = 32),
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    replaced_by bigint      REFERENCES refresh_tokens (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

-- +goose Down
DROP TABLE refresh_tokens;
DROP TABLE users;
