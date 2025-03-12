package database

const (
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