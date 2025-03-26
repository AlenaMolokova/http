package database

const (
	insertURLQuery = `
		INSERT INTO url_storage (short_id, original_url, user_id) 
		VALUES ($1, $2, $3) 
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

	selectByUserIDQuery = `
		SELECT short_id, original_url 
		FROM url_storage 
		WHERE user_id = $1
	`
)