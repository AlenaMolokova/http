package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	if dsn == "" {
		return nil, errors.New("пустая строка подключения к базе данных")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия соединения с базой данных: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}
	if err := createTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка создания таблиц: %v", err)
	}

	logrus.Info("Успешное подключение к базе данных PostgreSQL")
	return &PostgresStorage{db: db}, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS url_storage (
			short_id TEXT PRIMARY KEY,
			original_url TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			last_accessed_at TIMESTAMP WITH TIME ZONE 
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_original_url 
		ON url_storage(original_url);
	`)
	if err != nil {
		return fmt.Errorf("ошибка при создании таблицы: %v", err)
	}

	return nil
}

func (s *PostgresStorage) Save(shortID, originalURL string) error {
	query := `
		INSERT INTO url_storage (short_id, original_url) 
		VALUES ($1, $2) 
		ON CONFLICT (short_id) DO NOTHING
	`
	result, err := s.db.Exec(query, shortID, originalURL)
	if err !=nil{
		return err	
	}

	rowsAffected, err := result.RowsAffected()
	if err!=nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("url already exists")
	}

	return nil

}

func (s *PostgresStorage) Get(shortID string) (string, bool) {
	var originalURL string
	query := `
		SELECT original_url 
		FROM url_storage 
		WHERE short_id = $1 
	`

	err := s.db.QueryRow(query, shortID).Scan(&originalURL)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", false
		}
		logrus.WithError(err).Error("Ошибка получения URL")
		return "", false
	}

	return originalURL, true
}

func (s *PostgresStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.db.PingContext(ctx)
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

func (s *PostgresStorage) SaveBatch(items map[string]string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %v", err)
	}

	query := `
        INSERT INTO url_storage (short_id, original_url) 
        VALUES ($1, $2) 
        ON CONFLICT (short_id) DO NOTHING
    `

	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("ошибка при подготовке запроса: %v", err)
	}
	defer stmt.Close()

	for shortID, originalURL := range items {
		_, err := stmt.Exec(shortID, originalURL)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("ошибка при выполнении запроса: %v", err)
		}
	}

	return tx.Commit()
}

func (s *PostgresStorage) FindByOriginalURL(originalURL string) (string, error) {
	var shortID string
	query := `
	Select short_id
	FROM url_storage
	WHERE original_url = $1
	`

	err := s.db.QueryRow(query,originalURL).Scan(&shortID)
	if err !=nil {
		return "", err
	}
	return shortID, nil
}
