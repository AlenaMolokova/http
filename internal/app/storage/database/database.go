package database

import (
	"context"
	"fmt"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

type DatabaseStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(dsn string) (*DatabaseStorage, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	_, err = pool.Exec(context.Background(), CreateURLsTable)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create urls table: %w", err)
	}

	logrus.Info("Database storage initialized successfully")
	return &DatabaseStorage{pool: pool}, nil
}

func (db *DatabaseStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	_, err := db.pool.Exec(ctx, InsertURL, shortID, originalURL, userID)
	if err != nil {
		return fmt.Errorf("failed to save URL: %w", err)
	}
	return nil
}

func (db *DatabaseStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	var shortID string
	err := db.pool.QueryRow(ctx, SelectByOriginalURL, originalURL).Scan(&shortID)
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
	err := db.pool.QueryRow(ctx, SelectByShortID, shortID).Scan(&originalURL)
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
	rows, err := db.pool.Query(ctx, SelectByUserID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query URLs: %w", err)
	}
	defer rows.Close()

	var urls []models.UserURL
	for rows.Next() {
		var shortID, originalURL, userID string
		var isDeleted bool
		if err := rows.Scan(&shortID, &originalURL, &userID, &isDeleted); err != nil {
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
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for shortID, originalURL := range batch {
		_, err := tx.Exec(ctx, InsertURLBatch, shortID, originalURL, userID)
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
	if len(shortIDs) == 0 {
		return nil
	}
	_, err := db.pool.Exec(ctx, UpdateDeleteURLs, shortIDs, userID)
	if err != nil {
		return fmt.Errorf("failed to delete URLs: %w", err)
	}
	return nil
}

func (db *DatabaseStorage) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *DatabaseStorage) Close() error {
	db.pool.Close()
	return nil
}
