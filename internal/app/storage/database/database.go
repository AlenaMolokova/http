package database

import (
	"context"
	"database/sql"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.Exec(CreateURLsTable)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

func (ps *PostgresStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	_, err := ps.db.ExecContext(ctx, InsertURL, shortID, originalURL, userID)
	return err
}

func (ps *PostgresStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	var shortID string
	err := ps.db.QueryRowContext(ctx, SelectByOriginalURL, originalURL).Scan(&shortID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return shortID, nil
}

func (ps *PostgresStorage) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, InsertURLBatch)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for shortID, originalURL := range items {
		if _, err := stmt.ExecContext(ctx, shortID, originalURL, userID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (ps *PostgresStorage) Get(ctx context.Context, shortID string) (string, bool) {
	var originalURL string
	err := ps.db.QueryRowContext(ctx, SelectByShortID, shortID).Scan(&originalURL)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		logrus.WithError(err).Error("Failed to get URL")
		return "", false
	}
	return originalURL, true
}

func (ps *PostgresStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	rows, err := ps.db.QueryContext(ctx, SelectByUserID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []models.UserURL
	for rows.Next() {
		var url models.UserURL
		if err := rows.Scan(&url.ShortURL, &url.OriginalURL, &url.UserID, &url.IsDeleted); err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func (ps *PostgresStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	_, err := ps.db.ExecContext(ctx, UpdateDeleteURLs, shortIDs, userID)
	return err
}

func (ps *PostgresStorage) Ping(ctx context.Context) error {
	return ps.db.PingContext(ctx)
}