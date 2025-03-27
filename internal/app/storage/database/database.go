package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/lib/pq"
	"github.com/pressly/goose/v3"
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

	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка создания таблиц: %v", err)
	}

	logrus.Info("Успешное подключение к базе данных PostgreSQL")
	return &PostgresStorage{db: db}, nil
}

func applyMigrations(db *sql.DB) error {

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) Save(shortID, originalURL, userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка при начале транзакции: %v", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()

		if err != nil {
			logrus.WithError(err).Error("Ошибка коммита транзакции")
		} else {
			logrus.WithFields(logrus.Fields{
				"short_id":     shortID,
				"original_url": originalURL,
				"user_id":      userID,
			}).Info("URL успешно сохранён в базе данных")
		}
	}()

	result, err := tx.Exec(insertURLQuery, shortID, originalURL, userID)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка при проверке затронутых строк: %v", err)
	}

	if rowsAffected == 0 {
		logrus.WithField("short_id", shortID).Warn("URL с таким short_id уже существует")
		return errors.New("url with this short_id already exists")
	}

	return nil

}

func (s *PostgresStorage) Get(shortID string) (string, bool) {
	var originalURL string
    var isDeleted bool
    err := s.db.QueryRow(selectByShortIDQuery, shortID).Scan(&originalURL, &isDeleted)
    if err != nil {
        if err == sql.ErrNoRows {
            return "", false
        }
        logrus.WithError(err).Error("Ошибка получения URL")
        return "", false
    }
    if isDeleted {
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

func (s *PostgresStorage) SaveBatch(items map[string]string, userID string) error {
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
		_, err := stmt.Exec(shortID, originalURL, userID)
		if err != nil {
			return fmt.Errorf("ошибка при выполнении запроса: %v", err)
		}
	}

	return tx.Commit()

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

func (s *PostgresStorage) GetURLsByUserID(userID string) ([]models.UserURL, error) {
	var rows *sql.Rows
    var err error
    if userID == "" {
        rows, err = s.db.Query("SELECT short_id, original_url, is_deleted FROM url_storage")
    } else {
        rows, err = s.db.Query("SELECT short_id, original_url, is_deleted FROM url_storage WHERE user_id = $1", userID)
    }
    if err != nil {
        logrus.WithError(err).Error("Failed to query URLs by user ID")
        return nil, err
    }
    defer rows.Close()

    var urls []models.UserURL
    for rows.Next() {
        var url models.UserURL
        if err := rows.Scan(&url.ShortURL, &url.OriginalURL, &url.IsDeleted); err != nil {
            logrus.WithError(err).Error("Failed to scan URL row")
            return nil, err
        }
        urls = append(urls, url)
    }
    return urls, nil
}

func (s *PostgresStorage) MarkURLsAsDeleted(shortIDs []string, userID string) (int64, error) {
    result, err := s.db.Exec(updateDeletedQuery, pq.Array(shortIDs), userID)
    if err != nil {
        return 0, fmt.Errorf("ошибка при обновлении флага удаления: %v", err)
    }
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return 0, fmt.Errorf("ошибка при получении числа обновлённых строк: %v", err)
    }
    return rowsAffected, nil
}
