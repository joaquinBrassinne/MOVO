-- Businesses are retired (status = 'deleted'), never deleted: their counters are history
-- the business and the admin keep reading. With ON DELETE CASCADE, deleting a business
-- silently erased its metrics; RESTRICT makes that delete fail instead, so the rule holds
-- even for a path that forgets it. 00010 did the same for branches.

-- +goose Up
ALTER TABLE metric_daily DROP CONSTRAINT metric_daily_business_id_fkey;
ALTER TABLE metric_daily ADD CONSTRAINT metric_daily_business_id_fkey
    FOREIGN KEY (business_id) REFERENCES businesses (id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE metric_daily DROP CONSTRAINT metric_daily_business_id_fkey;
ALTER TABLE metric_daily ADD CONSTRAINT metric_daily_business_id_fkey
    FOREIGN KEY (business_id) REFERENCES businesses (id) ON DELETE CASCADE;
