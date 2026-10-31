-- Subscriptions (SDD 3, 4.1, 5.4). One row per business: the plan the super admin
-- confirmed and the date until which the business is paid. Mercado Pago is not called
-- by API in the MVP, so this row is the only record of what a business has paid for.
--
-- plan is set by the super admin when confirming the payment, never by the owner.
-- paid_until is a calendar date in Argentina time; a monthly payment extends it.
-- Every change is also written to audit_log (actor and reason), so the row holds the
-- current state and the audit log holds the history.

-- +goose Up
CREATE TABLE subscriptions (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id         bigint      NOT NULL UNIQUE REFERENCES businesses (id),
    plan                text        NOT NULL CHECK (plan IN ('starter', 'pro')),
    paid_until          date        NOT NULL,
    mp_subscription_ref text        CHECK (mp_subscription_ref IS NULL
                                           OR (btrim(mp_subscription_ref) <> '' AND char_length(mp_subscription_ref) <= 100)),
    confirmed_by        bigint      NOT NULL REFERENCES users (id),
    confirmed_at        timestamptz NOT NULL DEFAULT now(),
    notes               text        CHECK (notes IS NULL OR (btrim(notes) <> '' AND char_length(notes) <= 500)),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

-- "Por vencer" and expired lookups (5.6) scan by date.
CREATE INDEX subscriptions_paid_until_idx ON subscriptions (paid_until);

-- +goose Down
DROP TABLE subscriptions;
