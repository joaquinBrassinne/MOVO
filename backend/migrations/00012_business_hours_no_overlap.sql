-- Split shifts: a branch may open several times in one weekday, but two slots of the
-- same branch and weekday may never overlap. The exclusion constraint replaces the
-- UNIQUE (branch_id, weekday, opens_at) of 00003: two slots that start together
-- overlap, so it already covers that case. Slots touch ([09:00, 13:00) and [13:00,
-- 18:00)) without overlapping. They never cross midnight: CHECK (closes_at > opens_at)
-- from 00003 and the 23:59 limit of the application keep each slot inside one day,
-- and "closed" is simply a day with no slots.
--
-- Fails if a branch already stores overlapping slots; the application never allowed
-- them, so there should be none.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE business_hours DROP CONSTRAINT business_hours_branch_id_weekday_opens_at_key;

ALTER TABLE business_hours ADD CONSTRAINT business_hours_no_overlap EXCLUDE USING gist (
    branch_id WITH =,
    weekday WITH =,
    int4range(EXTRACT(EPOCH FROM opens_at)::int, EXTRACT(EPOCH FROM closes_at)::int) WITH &&
);

-- +goose Down
ALTER TABLE business_hours DROP CONSTRAINT business_hours_no_overlap;
ALTER TABLE business_hours ADD CONSTRAINT business_hours_branch_id_weekday_opens_at_key UNIQUE (branch_id, weekday, opens_at);
DROP EXTENSION btree_gist;
