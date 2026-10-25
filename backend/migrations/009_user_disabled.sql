-- +goose Up
ALTER TABLE users ADD COLUMN disabled_at TEXT;

-- +goose Down
-- SQLite cannot DROP COLUMN portably; leave disabled_at in place on down.
