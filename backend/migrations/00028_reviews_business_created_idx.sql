-- Reviews over time (SDD 6.16): the owner's metric averages the reviews of the whole business
-- in a period of up to 90 days. The existing indexes lead with branch_id (public lists) or
-- cover the moderation queue, so without this one the metric would read every review of the
-- platform. Hidden reviews are never counted, so they stay out of the index.

-- +goose Up
CREATE INDEX reviews_business_created_idx ON reviews (business_id, created_at) WHERE status <> 'hidden';

-- +goose Down
DROP INDEX reviews_business_created_idx;
