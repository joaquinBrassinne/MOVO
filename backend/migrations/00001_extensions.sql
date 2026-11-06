-- Enables PostGIS, required by the geography columns of phases 2 and 3.
-- The extension must be installed on the server (the development postgis/postgis
-- image ships it) and the role must be allowed to create it: in production the
-- migration runs with an administrative role.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;

-- +goose Down
-- The postgis/postgis image pre-installs postgis_topology and
-- postgis_tiger_geocoder, which depend on postgis and would block
-- `DROP EXTENSION postgis` with SQLSTATE 2BP01. They are dropped explicitly
-- instead of using CASCADE, which would also reach objects we know nothing
-- about: these three extensions are all an extension installation either way.
-- DROP EXTENSION IF EXISTS makes both statements no-ops on a server that never
-- had them.
DROP EXTENSION IF EXISTS postgis_tiger_geocoder;
DROP EXTENSION IF EXISTS postgis_topology;
DROP EXTENSION IF EXISTS postgis;
