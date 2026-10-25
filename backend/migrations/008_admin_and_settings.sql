-- +goose Up
ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;

-- Existing installs: promote the earliest user to admin.
UPDATE users SET is_admin = 1
WHERE id = (SELECT id FROM users ORDER BY created_at ASC LIMIT 1);

CREATE TABLE site_settings (
    key   TEXT PRIMARY KEY NOT NULL,
    value TEXT NOT NULL
);

INSERT INTO site_settings (key, value) VALUES ('registration_enabled', '1');

-- +goose Down
DROP TABLE IF EXISTS site_settings;
-- SQLite cannot DROP COLUMN portably; leave is_admin in place on down.
