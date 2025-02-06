package service

import (
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/storage"
)

type service struct {
	storage   storage.URLStorage
	generator generator.Generator
	baseURL   string
}

func NewURLService(storage storage.URLStorage, generator generator.Generator, baseURL string) URLService {
	return &service{
		storage:   storage,
		generator: generator,
		baseURL:   baseURL,
	}
}

func (s *service) ShortenURL(originalURL string) (string, error) {
	shortID := s.generator.Generate()
	if err := s.storage.Save(shortID, originalURL); err != nil {
		return "", err
	}
	return s.baseURL + "/" + shortID, nil
}

func (s *service) GetOriginalURL(shortID string) (string, bool) {
	return s.storage.Get(shortID)
}
