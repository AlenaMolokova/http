package database

const (
	createTableQuery = `
        CREATE TABLE IF NOT EXISTS url_storage (
            short_id VARCHAR(255) PRIMARY KEY,
            original_url TEXT NOT NULL,
            user_id VARCHAR(255) NOT NULL,
            is_deleted BOOLEAN NOT NULL DEFAULT FALSE
        )
    `
	insertURLQuery = `
        INSERT INTO url_storage (short_id, original_url, user_id, is_deleted) 
        VALUES ($1, $2, $3, FALSE) 
        ON CONFLICT (short_id) DO NOTHING
    `
	selectByShortIDQuery = `
        SELECT original_url, is_deleted 
        FROM url_storage 
        WHERE short_id = $1
    `
	selectByOriginalURLQuery = `
        SELECT short_id
        FROM url_storage
        WHERE original_url = $1 AND is_deleted = FALSE
    `
	selectByUserIDQuery = `
        SELECT short_id, original_url, is_deleted 
        FROM url_storage 
        WHERE user_id = $1
    `
	updateDeletedQuery = `
        UPDATE url_storage 
        SET is_deleted = TRUE 
        WHERE short_id = ANY($1) AND user_id = $2
        RETURNING short_id
    `
)