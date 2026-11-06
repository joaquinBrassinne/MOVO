-- Audit log: who did what to which entity. It is append-only: the trigger refuses
-- UPDATE and DELETE so the trail of super admin actions cannot be rewritten by the
-- application. There is no updated_at on purpose, an entry never changes.

-- +goose Up
CREATE TABLE audit_log (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_id   bigint      NOT NULL REFERENCES users (id),
    action     text        NOT NULL CHECK (btrim(action) <> ''),
    entity     text        NOT NULL CHECK (btrim(entity) <> ''),
    entity_id  bigint      NOT NULL,
    metadata   jsonb       NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(metadata) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_entity_idx ON audit_log (entity, entity_id, created_at DESC);

-- +goose StatementBegin
CREATE FUNCTION audit_log_is_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER audit_log_append_only
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_is_append_only();

-- +goose Down
DROP TRIGGER audit_log_append_only ON audit_log;
DROP FUNCTION audit_log_is_append_only();
DROP TABLE audit_log;
