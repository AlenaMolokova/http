package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createTableQuery)
	if err != nil {
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	_, err := s.db.ExecContext(ctx, insertURLQuery, shortID, originalURL, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil
		}
		return err
	}
	return nil
}

func (s *PostgresStorage) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	for shortID, originalURL := range items {
		_, err := s.db.ExecContext(ctx, insertURLQuery, shortID, originalURL, userID)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				continue
			}
			return err
		}
	}
	return nil
}

func (s *PostgresStorage) Get(ctx context.Context, shortID string) (string, bool) {
	var originalURL string
	var isDeleted bool
	err := s.db.QueryRowContext(ctx, selectByShortIDQuery, shortID).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false
		}
		logrus.WithError(err).Error("Failed to get URL by shortID")
		return "", false
	}
	if isDeleted {
		return "", false
	}
	return originalURL, true
}

func (s *PostgresStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	var shortID string
	err := s.db.QueryRowContext(ctx, selectByOriginalURLQuery, originalURL).Scan(&shortID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return shortID, nil
}

func (s *PostgresStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	rows, err := s.db.QueryContext(ctx, selectByUserIDQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []models.UserURL
	for rows.Next() {
		var shortID, originalURL string
		var isDeleted bool
		if err := rows.Scan(&shortID, &originalURL, &isDeleted); err != nil {
			return nil, err
		}
		urls = append(urls, models.UserURL{
			ShortURL:    shortID,
			OriginalURL: originalURL,
			IsDeleted:   isDeleted,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *PostgresStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	_, err := s.db.ExecContext(ctx, updateDeletedQuery, shortIDs, userID)
	return err
}