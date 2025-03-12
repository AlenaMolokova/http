-- +goose Up
CREATE TABLE IF NOT EXISTS url_storage (
    short_id TEXT PRIMARY KEY,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP WITH TIME ZONE 
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_original_url 
ON url_storage(original_url);

-- +goose Down
DROP TABLE IF EXISTS url_storage;