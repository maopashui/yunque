-- +goose Up
ALTER TABLE users ADD COLUMN avatar_color TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN avatar_color;
