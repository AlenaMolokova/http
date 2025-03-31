package file

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/AlenaMolokova/http/internal/app/models"
)

type FileStorage struct {
	filePath string
	urls     map[string]models.UserURL
	mu       sync.RWMutex
}

func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		filePath: filePath,
		urls:     make(map[string]models.UserURL),
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fs, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var entries []models.UserURL
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	for _, entry := range entries {
		fs.urls[entry.ShortURL] = entry
	}

	return fs, nil
}

func (s *FileStorage) writeToFile(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s.urls, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *FileStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[shortID] = models.UserURL{
		ShortURL:    shortID,
		OriginalURL: originalURL,
		UserID:      userID,
		IsDeleted:   false,
	}

	return s.writeToFile(ctx)
}

func (s *FileStorage) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
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

	return s.writeToFile(ctx)
}

func (s *FileStorage) Get(ctx context.Context, shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, exists := s.urls[shortID]
	if !exists || url.IsDeleted {
		return "", false
	}
	return url.OriginalURL, true
}

func (s *FileStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, url := range s.urls {
		if url.OriginalURL == originalURL && !url.IsDeleted {
			return url.ShortURL, nil
		}
	}
	return "", nil
}

func (s *FileStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var urls []models.UserURL
	for _, url := range s.urls {
		if url.UserID == userID {
			urls = append(urls, url)
		}
	}
	return urls, nil
}

func (s *FileStorage) Ping(ctx context.Context) error {
	return errors.New("file storage does not support database connection check")
}

func (s *FileStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, shortID := range shortIDs {
		if url, exists := s.urls[shortID]; exists && url.UserID == userID {
			url.IsDeleted = true
			s.urls[shortID] = url
		}
	}

	return s.writeToFile(ctx)
}