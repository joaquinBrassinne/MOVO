-- A hidden review keeps its text for a grace period and is then purged (SDD 5.5): the
-- comment and the name are erased and the row stays. The row has to stay because
-- reviews.benefit_code_id is UNIQUE: deleting it would let the customer review the same
-- coupon again and slip past the moderation that hid it.
--
-- hidden_at is when the super admin hid it (restoring clears it) and purged_at is when the
-- text was erased. A purged review cannot be restored: there is nothing left to show.

-- +goose Up
ALTER TABLE reviews
    ADD COLUMN hidden_at timestamptz,
    ADD COLUMN purged_at timestamptz;

-- Reviews hidden before this migration count their grace period from their last change.
UPDATE reviews SET hidden_at = updated_at WHERE status = 'hidden';

ALTER TABLE reviews
    ADD CONSTRAINT reviews_hidden_at_check CHECK ((status = 'hidden') = (hidden_at IS NOT NULL)),
    ADD CONSTRAINT reviews_purged_at_check CHECK (purged_at IS NULL OR status = 'hidden');

-- What the purge scans: hidden reviews whose text is still there, oldest first.
CREATE INDEX reviews_purge_idx ON reviews (hidden_at) WHERE status = 'hidden' AND purged_at IS NULL;

-- +goose Down
DROP INDEX reviews_purge_idx;
ALTER TABLE reviews
    DROP CONSTRAINT reviews_purged_at_check,
    DROP CONSTRAINT reviews_hidden_at_check,
    DROP COLUMN purged_at,
    DROP COLUMN hidden_at;
