package memory

import (
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
