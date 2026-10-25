-- Business catalog: categories, one business per owner, branches with a PostGIS
-- point (search runs per branch), opening hours and gallery images.
-- Slugs and WhatsApp numbers are stored already normalized by the application;
-- the checks make the database refuse anything else.

-- +goose Up
CREATE TABLE categories (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug       text        NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    name       text        NOT NULL CHECK (btrim(name) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO categories (slug, name) VALUES
    ('gimnasios', 'Gimnasios'),
    ('pilates', 'Pilates'),
    ('yoga', 'Yoga'),
    ('entrenadores', 'Entrenadores'),
    ('estetica', 'Estética'),
    ('natacion', 'Natación'),
    ('padel', 'Pádel');

CREATE TABLE businesses (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_user_id     bigint      NOT NULL UNIQUE REFERENCES users (id),
    category_id       bigint      NOT NULL REFERENCES categories (id),
    name              text        NOT NULL CHECK (btrim(name) <> ''),
    slug              text        NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    description       text        NOT NULL DEFAULT '',
    whatsapp_number   text        NOT NULL CHECK (whatsapp_number ~ '^[1-9][0-9]{7,14}$'),
    status            text        NOT NULL DEFAULT 'draft'
                                  CHECK (status IN ('draft', 'pending_review', 'active', 'paused', 'deleted')),
    pilot             boolean     NOT NULL DEFAULT false,
    verified_at       timestamptz,
    verified_by       bigint      REFERENCES users (id),
    payment_reference text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX businesses_category_id_idx ON businesses (category_id);
CREATE INDEX businesses_status_idx ON businesses (status);

CREATE TABLE branches (
    id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id          bigint        NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    name                 text          NOT NULL CHECK (btrim(name) <> ''),
    address              text          NOT NULL CHECK (btrim(address) <> ''),
    city                 text          NOT NULL CHECK (btrim(city) <> ''),
    location             geography(Point, 4326) NOT NULL,
    external_booking_url text,
    rating_avg           numeric(3, 2) NOT NULL DEFAULT 0 CHECK (rating_avg BETWEEN 0 AND 5),
    rating_count         integer       NOT NULL DEFAULT 0 CHECK (rating_count >= 0),
    active               boolean       NOT NULL DEFAULT true,
    created_at           timestamptz   NOT NULL DEFAULT now(),
    updated_at           timestamptz   NOT NULL DEFAULT now()
);

CREATE INDEX branches_business_id_idx ON branches (business_id);
CREATE INDEX branches_location_idx ON branches USING gist (location);

-- weekday: 0 = Sunday .. 6 = Saturday.
-- A branch may have several slots per day (split shifts) but never the same
-- opening time twice.
CREATE TABLE business_hours (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    branch_id  bigint      NOT NULL REFERENCES branches (id) ON DELETE CASCADE,
    weekday    smallint    NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    opens_at   time        NOT NULL,
    closes_at  time        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (closes_at > opens_at),
    UNIQUE (branch_id, weekday, opens_at)
);

CREATE TABLE business_images (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id bigint      NOT NULL REFERENCES businesses (id) ON DELETE CASCADE,
    object_key  text        NOT NULL UNIQUE CHECK (btrim(object_key) <> ''),
    position    integer     NOT NULL CHECK (position >= 0),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (business_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- +goose Down
DROP TABLE business_images;
DROP TABLE business_hours;
DROP TABLE branches;
DROP TABLE businesses;
DROP TABLE categories;
