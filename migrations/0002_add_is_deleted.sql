-- +goose Up
ALTER TABLE url_storage ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE url_storage DROP COLUMN is_deleted;