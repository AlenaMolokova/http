package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/AlenaMolokova/http/internal/app/models"
)

type MemoryStorage struct {
	urls map[string]models.UserURL
	mu   sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make(map[string]models.UserURL),
	}
}

func (s *MemoryStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[shortID] = models.UserURL{
		ShortURL:    shortID,
		OriginalURL: originalURL,
		UserID:      userID,
		IsDeleted:   false,
	}
	return nil
}

func (s *MemoryStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for shortID, url := range s.urls {
		if url.OriginalURL == originalURL && !url.IsDeleted {
			return shortID, nil
		}
	}
	return "", nil
}

func (s *MemoryStorage) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for shortID, originalURL := range items {
		s.urls[shortID] = models.UserURL{
			ShortURL:    shortID,
			OriginalURL: originalURL,
			UserID:      userID,
			IsDeleted:   false,
		}
	}
	return nil
}

func (s *MemoryStorage) Get(ctx context.Context, shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, exists := s.urls[shortID]
	if !exists || url.IsDeleted {
		return "", false
	}
	return url.OriginalURL, true
}

func (s *MemoryStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.UserURL
	for _, url := range s.urls {
		if url.UserID == userID && !url.IsDeleted {
			result = append(result, url)
		}
	}
	return result, nil
}

func (s *MemoryStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, shortID := range shortIDs {
		if url, exists := s.urls[shortID]; exists && url.UserID == userID {
			url.IsDeleted = true
			s.urls[shortID] = url
		}
	}
	return nil
}

func (s *MemoryStorage) Ping(ctx context.Context) error {
	return errors.New("memory storage does not support database connection check")
}
