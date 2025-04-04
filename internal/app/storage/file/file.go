package file

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
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
		logrus.Info("File does not exist, starting with empty storage")
		return fs, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		logrus.WithError(err).Error("Failed to read file")
		return nil, err
	}

	var entries []models.UserURL
	if err := json.Unmarshal(data, &entries); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal JSON from file")
		return nil, err
	}

	for _, entry := range entries {
		fs.urls[entry.ShortURL] = entry
	}

	logrus.Info("File storage initialized successfully")
	return fs, nil
}

func (fs *FileStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.urls[shortID] = models.UserURL{
		ShortURL:    shortID,
		OriginalURL: originalURL,
		UserID:      userID,
		IsDeleted:   false,
	}

	return fs.saveToFile()
}

func (fs *FileStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	for shortID, url := range fs.urls {
		if url.OriginalURL == originalURL && !url.IsDeleted {
			return shortID, nil
		}
	}
	return "", nil
}

func (fs *FileStorage) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for shortID, originalURL := range items {
		fs.urls[shortID] = models.UserURL{
			ShortURL:    shortID,
			OriginalURL: originalURL,
			UserID:      userID,
			IsDeleted:   false,
		}
	}

	return fs.saveToFile()
}

func (fs *FileStorage) Get(ctx context.Context, shortID string) (string, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	url, exists := fs.urls[shortID]
	if !exists || url.IsDeleted {
		return "", false
	}
	return url.OriginalURL, true
}

func (fs *FileStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var result []models.UserURL
	for _, url := range fs.urls {
		if url.UserID == userID && !url.IsDeleted {
			result = append(result, url)
		}
	}
	return result, nil
}

func (fs *FileStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, shortID := range shortIDs {
		if url, exists := fs.urls[shortID]; exists && url.UserID == userID {
			url.IsDeleted = true
			fs.urls[shortID] = url
		}
	}
	return fs.saveToFile()
}

func (fs *FileStorage) Ping(ctx context.Context) error {
	return errors.New("file storage does not support database connection check")
}

func (fs *FileStorage) saveToFile() error {
	var entries []models.UserURL
	for _, url := range fs.urls {
		entries = append(entries, url)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal URLs to JSON")
		return err
	}

	if err := os.WriteFile(fs.filePath, data, 0644); err != nil {
		logrus.WithError(err).Error("Failed to write URLs to file")
		return err
	}
	return nil
}