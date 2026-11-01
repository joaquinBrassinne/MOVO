-- Indexes for the business metrics (SDD 6.6): every metric reads the coupons of one business
-- over a period, by creation time or by redemption time, and "Tus récords" aggregates the
-- redemptions of the last 12 months. The existing indexes lead with (customer_id, status) and
-- (business_id, status), which cannot narrow a date range, so without these the 90-day and
-- monthly queries would read every coupon of the business.

-- +goose Up
CREATE INDEX benefit_codes_business_created_idx ON benefit_codes (business_id, created_at);

-- Only redeemed coupons have a redeemed_at, so the partial index stays small.
CREATE INDEX benefit_codes_business_redeemed_idx ON benefit_codes (business_id, redeemed_at)
    WHERE status = 'redeemed';

-- +goose Down
DROP INDEX benefit_codes_business_redeemed_idx;
DROP INDEX benefit_codes_business_created_idx;
