-- Daily aggregate counters (SDD 4.3): count, do not follow. A row is "how many times
-- this happened on this day"; nothing here identifies a person (no IP, no user agent,
-- no identifier, no path of a single visitor).
--
-- Profile views are per business, so profile_view has no branch and no service.
-- whatsapp_click carries its branch (and its service when the button came from one);
-- service_view carries both. The CHECK keeps that rule in the database.
--
-- NULLS NOT DISTINCT makes the unique key treat NULL branch and service as equal, so
-- the upsert adds to one row instead of creating a duplicate per NULL.

-- +goose Up
ALTER TABLE services ADD CONSTRAINT services_id_business_id_key UNIQUE (id, business_id);

CREATE TABLE metric_daily (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id bigint NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    branch_id   bigint,
    service_id  bigint,
    date        date   NOT NULL,
    metric      text   NOT NULL CHECK (metric IN ('profile_view', 'service_view', 'whatsapp_click')),
    count       bigint NOT NULL DEFAULT 0 CHECK (count >= 0),
    -- A branch or a service must belong to the same business. With a NULL column the
    -- composite key is not checked, which is what profile_view and the optional
    -- service of a click need.
    FOREIGN KEY (branch_id, business_id) REFERENCES branches (id, business_id) ON DELETE CASCADE,
    FOREIGN KEY (service_id, business_id) REFERENCES services (id, business_id) ON DELETE CASCADE,
    CONSTRAINT metric_daily_key UNIQUE NULLS NOT DISTINCT (business_id, branch_id, service_id, date, metric),
    CONSTRAINT metric_daily_shape CHECK (
        (metric = 'profile_view'   AND branch_id IS NULL AND service_id IS NULL)
     OR (metric = 'whatsapp_click' AND branch_id IS NOT NULL)
     OR (metric = 'service_view'   AND branch_id IS NOT NULL AND service_id IS NOT NULL)
    )
);

-- The panel reads one business over a date range.
CREATE INDEX metric_daily_business_date_idx ON metric_daily (business_id, date);

-- +goose Down
DROP TABLE metric_daily;
ALTER TABLE services DROP CONSTRAINT services_id_business_id_key;
