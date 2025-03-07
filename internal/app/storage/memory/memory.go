package memory

import (
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
)

type MemoryStorage struct {
	urls map[string]string
	mu   sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make(map[string]string),
	}
}

func (s *MemoryStorage) Save(shortID, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[shortID] = originalURL
	return nil
}

func (s *MemoryStorage) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.urls[shortID]
	logrus.WithFields(logrus.Fields{
		"shortID": shortID,
		"url":     url,
		"found":   ok,
	}).Info("Storage lookup")
	return url, ok
}

func (s *MemoryStorage) Ping() error {
	return errors.New("memory storage does not support database connection check")
}

func (s *MemoryStorage) SaveBatch(items map[string]string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    for shortID, originalURL := range items {
        s.urls[shortID] = originalURL
    }
    
    return nil
}
