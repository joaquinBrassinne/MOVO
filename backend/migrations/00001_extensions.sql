-- Enables PostGIS, required by the geography columns of phases 2 and 3.
-- The extension must be installed on the server (the development postgis/postgis
-- image ships it) and the role must be allowed to create it: in production the
-- migration runs with an administrative role.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;

-- +goose Down
DROP EXTENSION IF EXISTS postgis;
