-- Services: what a business offers (a yoga class, a court rental) and when. A
-- service is tied to one branch or, without branch, to the whole business.
-- The composite foreign key makes the database refuse a branch of another business.

-- +goose Up
ALTER TABLE branches ADD CONSTRAINT branches_id_business_id_key UNIQUE (id, business_id);

CREATE TABLE services (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id bigint      NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    branch_id   bigint,
    name        text        NOT NULL CHECK (btrim(name) <> ''),
    description text        NOT NULL DEFAULT '',
    active      boolean     NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    -- MATCH SIMPLE: with branch_id NULL the constraint is not checked.
    FOREIGN KEY (branch_id, business_id) REFERENCES branches (id, business_id) ON DELETE CASCADE
);

CREATE INDEX services_business_id_idx ON services (business_id);

-- weekday: 0 = Sunday .. 6 = Saturday. Several classes per day are allowed, never
-- two starting at the same time.
CREATE TABLE service_schedules (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    service_id bigint      NOT NULL REFERENCES services (id) ON DELETE CASCADE,
    weekday    smallint    NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    starts_at  time        NOT NULL,
    ends_at    time        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at),
    UNIQUE (service_id, weekday, starts_at)
);

-- +goose Down
DROP TABLE service_schedules;
DROP TABLE services;
ALTER TABLE branches DROP CONSTRAINT branches_id_business_id_key;
