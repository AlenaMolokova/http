package memory

import (
	"errors"
	"sync"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
)

type URLRecord struct {
	ShortID     string
	OriginalURL string
	UserID      string
	IsDeleted   bool
}

type MemoryStorage struct {
	urls map[string]URLRecord
	mu   sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make(map[string]URLRecord),
	}
}

func (s *MemoryStorage) Save(shortID, originalURL, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[shortID] = URLRecord{
		ShortID:     shortID,
		OriginalURL: originalURL,
		UserID:      userID,
		IsDeleted:   false,
	}
	return nil
}

func (s *MemoryStorage) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.urls[shortID]
	logrus.WithFields(logrus.Fields{
		"shortID": shortID,
		"url":     record.OriginalURL,
		"found":   ok,
	}).Info("Storage lookup")

	if !ok || record.IsDeleted {
		return "", false
	}

	return record.OriginalURL, true
}

func (s *MemoryStorage) Ping() error {
	return errors.New("memory storage does not support database connection check")
}

func (s *MemoryStorage) SaveBatch(items map[string]string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for shortID, originalURL := range items {
		s.urls[shortID] = URLRecord{
			ShortID:     shortID,
			OriginalURL: originalURL,
			UserID:      userID,
			IsDeleted:   false,
		}
	}

	return nil
}

func (s *MemoryStorage) FindByOriginalURL(originalURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for shortID, record := range s.urls {
		if record.OriginalURL == originalURL && !record.IsDeleted {
			return shortID, nil
		}
	}

	return "", errors.New("url not found")
}

func (s *MemoryStorage) GetURLsByUserID(userID string) ([]models.UserURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.UserURL

	for _, record := range s.urls {
		if record.UserID == userID && !record.IsDeleted {
			result = append(result, models.UserURL{
				ShortURL:    record.ShortID,
				OriginalURL: record.OriginalURL,
				IsDeleted:   record.IsDeleted,
			})
		}
	}

	return result, nil
}

func (s *MemoryStorage) MarkURLsAsDeleted(shortIDs []string, userID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var rowsAffected int64

	for _, shortID := range shortIDs {
		if record, ok := s.urls[shortID]; ok && record.UserID == userID {
			record.IsDeleted = true
			s.urls[shortID] = record
			rowsAffected++
		}
	}

	logrus.WithFields(logrus.Fields{
		"short_ids":     shortIDs,
		"user_id":       userID,
		"rows_affected": rowsAffected,
	}).Info("URLs marked as deleted in memory storage")
	return rowsAffected, nil
}
