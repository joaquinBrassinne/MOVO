-- Objects that are no longer referenced and still have to be deleted from the object
-- storage. A removed photo queues its key here in the same transaction that deletes its
-- row, so a storage that is down (or a crash) at that moment leaves the object queued
-- instead of lost; the delete is retried until it works and then the row is dropped.

-- +goose Up
CREATE TABLE object_cleanup (
    object_key      text        PRIMARY KEY CHECK (btrim(object_key) <> ''),
    queued_at       timestamptz NOT NULL DEFAULT now(),
    attempts        integer     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error      text,
    last_attempt_at timestamptz
);

-- +goose Down
DROP TABLE object_cleanup;
