-- Assisted password reset (SDD 6.2): the super admin issues a one-time link for an owner and
-- sends it by WhatsApp. Only the SHA-256 digest of the token is stored, so a leaked database
-- cannot be replayed. A token is valid until expires_at and works once: used_at is set when it
-- is redeemed or when a newer link for the same user supersedes it.

-- +goose Up
CREATE TABLE password_reset_tokens (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users (id),
    token_hash bytea       NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Issuing a new link retires the unused ones of that user.
CREATE INDEX password_reset_tokens_pending_idx ON password_reset_tokens (user_id) WHERE used_at IS NULL;

-- +goose Down
DROP TABLE password_reset_tokens;
