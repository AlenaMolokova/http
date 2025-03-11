package database

const (
	createTableQuery = `
		CREATE TABLE IF NOT EXISTS url_storage (
			short_id TEXT PRIMARY KEY,
			original_url TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			last_accessed_at TIMESTAMP WITH TIME ZONE 
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_original_url 
		ON url_storage(original_url);
	`

	insertURLQuery = `
		INSERT INTO url_storage (short_id, original_url) 
		VALUES ($1, $2) 
		ON CONFLICT (short_id) DO NOTHING
	`

	selectByShortIDQuery = `
		SELECT original_url 
		FROM url_storage 
		WHERE short_id = $1 
	`

	selectByOriginalURLQuery = `
		SELECT short_id
		FROM url_storage
		WHERE original_url = $1
	`
)