package service

import (
    "github.com/AlenaMolokova/http/internal/app/generator"
)

type URLStorage interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, bool)
	Ping() error
}

type URLService interface {
	ShortenURL(originalURL string) (string, error)
	GetOriginalURL(shortID string) (string, bool)
	Ping() error
}

type service struct {
	storage   URLStorage
	generator generator.Generator
	baseURL   string
}

func NewURLService(storage URLStorage, generator generator.Generator, baseURL string) URLService {
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
func (s *service) Ping() error{
	return s.storage.Ping()
}
