-- The monthly coupon quota belongs to the business, not to each benefit (SDD 5.2): the
-- owner sets one cap for every coupon the business generates in a calendar month
-- (America/Argentina/Cordoba). NULL means no limit, so a business starts without one.
--
-- The per-benefit limits of 00014 are dropped without migrating their values: they
-- counted something else, and the owners that exist today have no limit until they set one.

-- +goose Up
ALTER TABLE businesses ADD COLUMN monthly_coupon_limit integer CHECK (monthly_coupon_limit > 0);
ALTER TABLE benefits DROP COLUMN monthly_limit;

-- +goose Down
ALTER TABLE benefits ADD COLUMN monthly_limit integer;
-- A benefit gets the limit of its business, or a number no business reaches when it has none.
UPDATE benefits SET monthly_limit = COALESCE(
    (SELECT monthly_coupon_limit FROM businesses WHERE businesses.id = benefits.business_id), 100000);
ALTER TABLE benefits ALTER COLUMN monthly_limit SET NOT NULL;
ALTER TABLE benefits ADD CONSTRAINT benefits_monthly_limit_check CHECK (monthly_limit > 0);
ALTER TABLE businesses DROP COLUMN monthly_coupon_limit;
