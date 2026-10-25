-- Branches are retired (active = false), never deleted: metrics, coupon redemptions and
-- reviews point at them and are history the business and the admin keep reading. With
-- ON DELETE CASCADE, deleting a branch silently erased its counters; RESTRICT makes
-- that delete fail instead, so the rule holds even for a path that forgets it.

-- +goose Up
ALTER TABLE metric_daily DROP CONSTRAINT metric_daily_branch_id_business_id_fkey;
ALTER TABLE metric_daily ADD CONSTRAINT metric_daily_branch_id_business_id_fkey
    FOREIGN KEY (branch_id, business_id) REFERENCES branches (id, business_id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE metric_daily DROP CONSTRAINT metric_daily_branch_id_business_id_fkey;
ALTER TABLE metric_daily ADD CONSTRAINT metric_daily_branch_id_business_id_fkey
    FOREIGN KEY (branch_id, business_id) REFERENCES branches (id, business_id) ON DELETE CASCADE;
