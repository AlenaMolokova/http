package database

import (
	"context"
	"fmt"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
)

type DatabaseStorage struct {
	conn *pgx.Conn
}

func NewPostgresStorage(dsn string) (*DatabaseStorage, error) { 
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &DatabaseStorage{conn: conn}, nil
}

func (db *DatabaseStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	_, err := db.conn.Exec(ctx, "INSERT INTO urls (short_id, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING", shortID, originalURL, userID)
	if err != nil {
		return fmt.Errorf("failed to save URL: %w", err)
	}
	return nil
}

func (db *DatabaseStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	var shortID string
	err := db.conn.QueryRow(ctx, "SELECT short_id FROM urls WHERE original_url = $1 AND deleted = false", originalURL).Scan(&shortID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to find URL: %w", err)
	}
	return shortID, nil
}

func (db *DatabaseStorage) Get(ctx context.Context, shortID string) (string, bool) {
	var originalURL string
	err := db.conn.QueryRow(ctx, "SELECT original_url FROM urls WHERE short_id = $1 AND deleted = false", shortID).Scan(&originalURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false
		}
		logrus.WithError(err).Error("Failed to get URL")
		return "", false
	}
	return originalURL, true
}

func (db *DatabaseStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	rows, err := db.conn.Query(ctx, "SELECT short_id, original_url FROM urls WHERE user_id = $1 AND deleted = false", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query URLs: %w", err)
	}
	defer rows.Close()

	var urls []models.UserURL
	for rows.Next() {
		var shortID, originalURL string
		if err := rows.Scan(&shortID, &originalURL); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		urls = append(urls, models.UserURL{ShortURL: shortID, OriginalURL: originalURL})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return urls, nil
}

func (db *DatabaseStorage) SaveBatch(ctx context.Context, batch map[string]string, userID string) error {
	tx, err := db.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for shortID, originalURL := range batch {
		_, err := tx.Exec(ctx, "INSERT INTO urls (short_id, original_url, user_id) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING", shortID, originalURL, userID)
		if err != nil {
			return fmt.Errorf("failed to save batch URL: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (db *DatabaseStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	_, err := db.conn.Exec(ctx, "UPDATE urls SET deleted = true WHERE short_id = ANY($1) AND user_id = $2", shortIDs, userID)
	if err != nil {
		return fmt.Errorf("failed to delete URLs: %w", err)
	}
	return nil
}

func (db *DatabaseStorage) Ping(ctx context.Context) error {
	return db.conn.Ping(ctx)
}

func (db *DatabaseStorage) Close() error {
	return db.conn.Close(context.Background())
}