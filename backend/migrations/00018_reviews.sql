-- Reviews verified by redemption (SDD 5.5). Only a customer who redeemed a coupon can
-- review, once per coupon, at the branch that redeemed it. The composite foreign key to
-- (id, redeemed_branch_id) makes the database refuse a review of a coupon that is not
-- redeemed or that belongs to another branch.
--
-- Reviews are history: nothing they point at cascades away, the owner cannot delete
-- them and moderation hides them (status). A data-erasure request anonymizes
-- display_name instead of deleting the row. Whether the comment stays free of phones and
-- URLs is the domain's job, as for benefits.
--
-- branches.rating_avg and rating_count already exist since 00003; the domain event
-- handler (4C.5) keeps them in step with this table.

-- +goose Up
ALTER TABLE benefit_codes
    ADD CONSTRAINT benefit_codes_id_redeemed_branch_key UNIQUE (id, redeemed_branch_id);

CREATE TABLE reviews (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    branch_id       bigint      NOT NULL,
    business_id     bigint      NOT NULL REFERENCES businesses (id),
    benefit_code_id bigint      NOT NULL UNIQUE,
    customer_id     bigint      NOT NULL REFERENCES customers (id),
    rating          smallint    NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment         text        CHECK (comment IS NULL OR (btrim(comment) <> '' AND char_length(comment) <= 500)),
    display_name    text        NOT NULL CHECK (btrim(display_name) <> ''),
    show_name       boolean     NOT NULL DEFAULT true,
    status          text        NOT NULL DEFAULT 'visible' CHECK (status IN ('visible', 'reported', 'hidden')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (branch_id, business_id) REFERENCES branches (id, business_id),
    FOREIGN KEY (benefit_code_id, branch_id) REFERENCES benefit_codes (id, redeemed_branch_id)
);

-- The public listing of a branch, newest first.
CREATE INDEX reviews_branch_visible_idx ON reviews (branch_id, created_at DESC, id DESC) WHERE status = 'visible';
-- The moderation queue.
CREATE INDEX reviews_moderation_idx ON reviews (created_at DESC) WHERE status <> 'visible';

-- +goose Down
DROP TABLE reviews;
ALTER TABLE benefit_codes DROP CONSTRAINT benefit_codes_id_redeemed_branch_key;
