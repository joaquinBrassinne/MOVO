-- Two optional details the owner panel shows: an estimated price on each service (whole
-- pesos, NULL when the owner does not set one) and a short reference on each branch
-- (floor, door bell, "al fondo"). Neither changes how anything is searched or limited.

-- +goose Up
ALTER TABLE services ADD COLUMN price integer CHECK (price >= 0 AND price <= 100000000);
ALTER TABLE branches ADD COLUMN reference text NOT NULL DEFAULT '' CHECK (char_length(reference) <= 100);

-- +goose Down
ALTER TABLE branches DROP COLUMN reference;
ALTER TABLE services DROP COLUMN price;
