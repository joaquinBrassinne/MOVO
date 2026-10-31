-- visible_branches learns about payments (SDD 4.1, task 5.4). A branch is public when it
-- is active, its business is active and the business is a pilot or paid: today is on or
-- before paid_until plus the grace period.
--
-- The columns do not change (CREATE OR REPLACE VIEW cannot reorder them and the search
-- queries read them by name). featured now follows the plan: only a paying Pro business is
-- featured; a pilot gets Pro limits for free but is not promoted.
--
-- Two numbers live here because a view cannot read configuration: the grace days (7, the
-- same as billing/domain.GraceDays; a test pins that they agree) and the time zone, so that
-- "today" is the Argentine calendar day, as in the domain.

-- +goose Up
CREATE OR REPLACE VIEW visible_branches AS
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
       COALESCE(s.plan = 'pro', false) AS featured
FROM branches br
JOIN businesses b ON b.id = br.business_id
LEFT JOIN subscriptions s ON s.business_id = b.id
WHERE br.active
  AND b.status = 'active'
  AND (b.pilot
       OR (s.id IS NOT NULL
           AND (now() AT TIME ZONE 'America/Argentina/Buenos_Aires')::date <= s.paid_until + 7));

-- +goose Down
CREATE OR REPLACE VIEW visible_branches AS
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
