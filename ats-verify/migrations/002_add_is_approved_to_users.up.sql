-- +goose Up
ALTER TABLE users ADD COLUMN is_approved BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users DROP COLUMN is_approved;
