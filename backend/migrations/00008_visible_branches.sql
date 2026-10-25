-- visible_branches is the only place that decides which branches the public can find
-- (SDD 4.1). This is the minimal version used while every approved business is visible:
-- an active branch of a business whose status is 'active'. Phase 5 replaces it with
-- CREATE OR REPLACE VIEW, adding the pilot / paid-until-plus-grace condition, so the
-- columns below must not change. featured is always false until the subscription
-- (and its plan) exists.

-- +goose Up
CREATE VIEW visible_branches AS
SELECT br.id            AS branch_id,
       b.id             AS business_id,
       b.category_id,
       b.name           AS business_name,
       b.slug           AS business_slug,
       br.name          AS branch_name,
       br.address,
       br.city,
       br.reference,
       br.location,
       br.rating_avg,
       br.rating_count,
       b.verified_at,
       false            AS featured
FROM branches br
JOIN businesses b ON b.id = br.business_id
WHERE br.active
  AND b.status = 'active';

-- +goose Down
DROP VIEW visible_branches;
