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
	_, err := db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("ошибка при создании таблицы: %v", err)
	}

	return nil
}

func (s *PostgresStorage) Save(shortID, originalURL string) error {
	result, err := s.db.Exec(insertURLQuery, shortID, originalURL)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("url already exists")
	}

	return nil

}

func (s *PostgresStorage) Get(shortID string) (string, bool) {
	var originalURL string

	err := s.db.QueryRow(selectByShortIDQuery, shortID).Scan(&originalURL)

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
	const batchSize = 1000
	itemCount := len(items)

	var tx *sql.Tx
	var err error

	if itemCount <= batchSize {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("ошибка при начале транзакции: %v", err)
		}

		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()

		stmt, err := tx.Prepare(insertURLQuery)
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

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("ошибка при фиксации транзакции: %v", err)
		}

		return nil
	}
	processed := 0
	batch := make(map[string]string)

	for shortID, originalURL := range items {
		batch[shortID] = originalURL
		processed++

		if len(batch) >= batchSize || processed == itemCount {
			tx, err = s.db.Begin()
			if err != nil {
				return fmt.Errorf("ошибка при начале транзакции: %v", err)
			}
		}

		stmt, err := tx.Prepare(insertURLQuery)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("ошибка при подготовке запроса: %v", err)
		}

		for batchShortID, batchOriginalURL := range batch {
			_, err := stmt.Exec(batchShortID, batchOriginalURL)
			if err != nil {
				stmt.Close()
				tx.Rollback()
				return fmt.Errorf("ошибка при выполнении запроса: %v", err)
			}
		}

		stmt.Close()
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("ошибка при фиксации транзакции: %v", err)
		}

		batch = make(map[string]string)
	}

	return nil
}

func (s *PostgresStorage) FindByOriginalURL(originalURL string) (string, error) {
	var shortID string

	err := s.db.QueryRow(selectByOriginalURLQuery, originalURL).Scan(&shortID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("url not found")
		}
		return "", fmt.Errorf("ошибка при поиске URL: %v", err)
	}
	return shortID, nil
}
