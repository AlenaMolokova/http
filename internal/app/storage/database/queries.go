package database

// SQL-запросы для работы с базой данных URL-сокращений

// CreateURLsTable - SQL-запрос для создания таблицы urls, если она не существует.
// Таблица содержит следующие поля:
//   - short_id: первичный ключ, сокращенный идентификатор URL
//   - original_url: оригинальный URL-адрес
//   - user_id: идентификатор пользователя, создавшего сокращение
//   - is_deleted: флаг, указывающий, удален ли URL
const (
	CreateURLsTable = `
		CREATE TABLE IF NOT EXISTS urls (
			short_id VARCHAR(255) PRIMARY KEY,
			original_url TEXT NOT NULL,
			user_id VARCHAR(255),
			is_deleted BOOLEAN DEFAULT FALSE
		)`

	// InsertURL - SQL-запрос для добавления нового URL в базу данных.
	// При конфликте первичного ключа (short_id) запись не добавляется.
	// Параметры:
	//   - $1: сокращенный идентификатор (short_id)
	//   - $2: оригинальный URL-адрес (original_url)
	//   - $3: идентификатор пользователя (user_id)
	InsertURL = `
		INSERT INTO urls (short_id, original_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (short_id) DO NOTHING`

	// SelectByOriginalURL - SQL-запрос для поиска сокращенного идентификатора по оригинальному URL.
	// Возвращает только неудаленные URL-адреса.
	// Параметры:
	//   - $1: оригинальный URL-адрес (original_url)
	SelectByOriginalURL = `
		SELECT short_id
		FROM urls
		WHERE original_url = $1 AND is_deleted = FALSE
		LIMIT 1`

	// InsertURLBatch - SQL-запрос для пакетного добавления URL-адресов.
	// При конфликте первичного ключа (short_id) запись не добавляется.
	// Параметры:
	//   - $1: сокращенный идентификатор (short_id)
	//   - $2: оригинальный URL-адрес (original_url)
	//   - $3: идентификатор пользователя (user_id)
	InsertURLBatch = `
		INSERT INTO urls (short_id, original_url, user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (short_id) DO NOTHING`

	// SelectByShortID - SQL-запрос для получения оригинального URL по сокращенному идентификатору.
	// Возвращает только неудаленные URL-адреса.
	// Параметры:
	//   - $1: сокращенный идентификатор (short_id)
	SelectByShortID = `
		SELECT original_url
		FROM urls
		WHERE short_id = $1 AND is_deleted = FALSE`

	// SelectByUserID - SQL-запрос для получения всех URL-адресов, созданных определенным пользователем.
	// Возвращает только неудаленные URL-адреса.
	// Параметры:
	//   - $1: идентификатор пользователя (user_id)
	SelectByUserID = `
		SELECT short_id, original_url, user_id, is_deleted
		FROM urls
		WHERE user_id = $1 AND is_deleted = FALSE`

	// UpdateDeleteURLs - SQL-запрос для пометки URL-адресов как удаленных.
	// Обновляет только те URL-адреса, которые принадлежат указанному пользователю.
	// Параметры:
	//   - $1: массив сокращенных идентификаторов (short_id)
	//   - $2: идентификатор пользователя (user_id)
	UpdateDeleteURLs = `
		UPDATE urls
		SET is_deleted = TRUE
		WHERE short_id = ANY($1) AND user_id = $2`
)