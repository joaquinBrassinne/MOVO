-- Services are deleted softly: metric_daily points at them and that history is kept, like
-- the branches' (00010). With ON DELETE CASCADE, deleting a service silently erased its
-- counters; RESTRICT makes a physical delete fail instead, so the rule holds even for a
-- path that forgets it. The app sets deleted_at and every owner-side read skips those rows.

-- +goose Up
ALTER TABLE services ADD COLUMN deleted_at timestamptz;

ALTER TABLE metric_daily DROP CONSTRAINT metric_daily_service_id_business_id_fkey;
ALTER TABLE metric_daily ADD CONSTRAINT metric_daily_service_id_business_id_fkey
    FOREIGN KEY (service_id, business_id) REFERENCES services (id, business_id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE metric_daily DROP CONSTRAINT metric_daily_service_id_business_id_fkey;
ALTER TABLE metric_daily ADD CONSTRAINT metric_daily_service_id_business_id_fkey
    FOREIGN KEY (service_id, business_id) REFERENCES services (id, business_id) ON DELETE CASCADE;

-- The down schema has no soft delete: the deleted rows go, and their counters with them.
DELETE FROM services WHERE deleted_at IS NOT NULL;
ALTER TABLE services DROP COLUMN deleted_at;
