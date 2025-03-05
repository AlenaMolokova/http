package database

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type DBStorage struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	if dsn == "" {
		return nil, errors.New("пустая строка подключения к базе данных")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка при проверке соединения с базой данных: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			short_id TEXT UNIQUE NOT NULL,
			original_url TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка при создании таблицы urls: %v", err)
	}

	logrus.Info("Успешное подключение к базе данных PostgreSQL")

	return &DBStorage{
		db: db,
	}, nil
}

func (s *DBStorage) Ping() error {
	return s.db.Ping()
}

func (s *DBStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *DBStorage) Save(shortID, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO urls (short_id, original_url)
		VALUES ($1, $2)
		ON CONFLICT (short_id)
		DO NOTHING
	`, shortID, originalURL)

	if err != nil {
		return fmt.Errorf("ошибка при сохранении URL в базу данных: %v", err)
	}

	return nil
}

func (s *DBStorage) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var originalURL string
	err := s.db.QueryRow(`
		SELECT original_url FROM urls
		WHERE short_id = $1
	`, shortID).Scan(&originalURL)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logrus.WithField("shortID", shortID).Info("URL не найден в базе данных")
			return "", false
		}
		logrus.WithError(err).WithField("shortID", shortID).Error("Ошибка при получении URL из базы данных")
		return "", false
	}

	logrus.WithFields(logrus.Fields{
		"shortID": shortID,
		"url":     originalURL,
	}).Info("URL найден в базе данных")

	return originalURL, true
}