-- Benefits and coupons (SDD 4, 5.2). A customer has no account: only the name and the
-- WhatsApp number they declare. The phone is stored already normalized (digits with
-- country code) and is unique, so the same person cannot slip past the "one pending
-- coupon per customer and business" rule by being inserted twice.
--
-- A benefit is a percent discount (10/15/20) or a welcome gift; the CHECK keeps each
-- type's fields coherent. benefit_codes keeps the history of every coupon, so it never
-- cascades away: a benefit or a business with coupons cannot be deleted.
--
-- Redeeming is one atomic UPDATE ... WHERE status = 'pending' (SDD 5.2); the CHECKs
-- below make the database refuse a redeemed coupon without who/where/when.

-- +goose Up
CREATE TABLE customers (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       text        NOT NULL CHECK (btrim(name) <> ''),
    phone      text        NOT NULL UNIQUE CHECK (phone ~ '^[1-9][0-9]{7,14}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE benefits (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id      bigint      NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    type             text        NOT NULL CHECK (type IN ('percent', 'welcome_gift')),
    percent          smallint,
    gift_description text,
    active           boolean     NOT NULL DEFAULT true,
    monthly_limit    integer     NOT NULL CHECK (monthly_limit > 0),
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT benefits_id_business_id_key UNIQUE (id, business_id),
    CONSTRAINT benefits_shape CHECK (
        -- IS NOT NULL is explicit: a NULL operand makes the whole branch NULL, and a
        -- CHECK that evaluates to NULL passes.
        (type = 'percent'      AND percent IS NOT NULL AND percent IN (10, 15, 20) AND gift_description IS NULL)
     OR (type = 'welcome_gift' AND percent IS NULL AND gift_description IS NOT NULL AND btrim(gift_description) <> '')
    )
);

CREATE INDEX benefits_business_id_idx ON benefits (business_id);

CREATE TABLE benefit_codes (
    id                 bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code               text        NOT NULL UNIQUE CHECK (code ~ '^[A-HJ-NP-Z2-9]{8}$'),
    benefit_id         bigint      NOT NULL,
    business_id        bigint      NOT NULL REFERENCES businesses (id),
    customer_id        bigint      NOT NULL REFERENCES customers (id),
    access_token_hash  bytea       NOT NULL UNIQUE CHECK (length(access_token_hash) = 32),
    status             text        NOT NULL DEFAULT 'pending'
                                   CHECK (status IN ('pending', 'redeemed', 'expired')),
    expires_at         timestamptz NOT NULL,
    redeemed_at        timestamptz,
    redeemed_branch_id bigint,
    redeemed_by        bigint      REFERENCES users (id),
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    -- The benefit and the redeeming branch must belong to the coupon's business.
    FOREIGN KEY (benefit_id, business_id) REFERENCES benefits (id, business_id),
    FOREIGN KEY (redeemed_branch_id, business_id) REFERENCES branches (id, business_id),
    CONSTRAINT benefit_codes_redeemed_shape CHECK (
        (status = 'redeemed' AND redeemed_at IS NOT NULL AND redeemed_by IS NOT NULL AND redeemed_branch_id IS NOT NULL)
     OR (status <> 'redeemed' AND redeemed_at IS NULL AND redeemed_by IS NULL AND redeemed_branch_id IS NULL)
    )
);

CREATE INDEX benefit_codes_customer_status_idx ON benefit_codes (customer_id, status);
CREATE INDEX benefit_codes_business_status_idx ON benefit_codes (business_id, status);

-- One pending coupon per customer and business (SDD 5.2). A redeemed or expired
-- coupon frees the slot.
CREATE UNIQUE INDEX benefit_codes_one_pending_idx ON benefit_codes (customer_id, business_id) WHERE status = 'pending';

-- +goose Down
DROP TABLE benefit_codes;
DROP TABLE benefits;
DROP TABLE customers;
