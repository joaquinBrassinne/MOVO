-- Alerts (SDD 3): notices the platform leaves for a business owner. The first one is the
-- plan downgrade (SDD 4.2): what the new plan no longer fits was switched off and the owner
-- must be told why. Phase 6 reuses the table for the Pro "coupons not redeemed" alert.
--
-- payload carries what the owner's panel needs to explain the notice, so the alert still
-- makes sense after the things it mentions change. read_at is set when the owner dismisses it.

-- +goose Up
CREATE TABLE alerts (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id bigint      NOT NULL REFERENCES businesses (id),
    type        text        NOT NULL CHECK (btrim(type) <> ''),
    payload     jsonb       NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(payload) = 'object'),
    created_at  timestamptz NOT NULL DEFAULT now(),
    read_at     timestamptz
);

-- The owner's panel reads the unread ones, newest first.
CREATE INDEX alerts_unread_idx ON alerts (business_id, created_at DESC, id DESC) WHERE read_at IS NULL;

-- +goose Down
DROP TABLE alerts;
