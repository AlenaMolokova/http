package database

const (
	CreateURLsTable = `
		CREATE TABLE IF NOT EXISTS urls (
			short_id VARCHAR(255) PRIMARY KEY,
			original_url TEXT NOT NULL,
			user_id VARCHAR(255),
			is_deleted BOOLEAN DEFAULT FALSE
		)`

	InsertURL = `
		INSERT INTO urls (short_id, original_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (short_id) DO NOTHING`

	SelectByOriginalURL = `
		SELECT short_id
		FROM urls
		WHERE original_url = $1 AND is_deleted = FALSE
		LIMIT 1`

	InsertURLBatch = `
		INSERT INTO urls (short_id, original_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (short_id) DO NOTHING`

	SelectByShortID = `
		SELECT original_url
		FROM urls
		WHERE short_id = $1 AND is_deleted = FALSE`

	SelectByUserID = `
		SELECT short_id, original_url, user_id, is_deleted
		FROM urls
		WHERE user_id = $1 AND is_deleted = FALSE`

	UpdateDeleteURLs = `
		UPDATE urls
		SET is_deleted = TRUE
		WHERE short_id = ANY($1) AND user_id = $2`
)
