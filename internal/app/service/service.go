package service

import (
    "github.com/AlenaMolokova/http/internal/app/generator"
    "github.com/AlenaMolokova/http/internal/app/storage"
)

type URLService struct {
    storage   storage.URLStorage
    generator generator.URLGenerator
    baseURL   string
}

func NewURLService(storage storage.URLStorage, generator generator.URLGenerator, baseURL string) *URLService {
    return &URLService{
        storage:   storage,
        generator: generator,
        baseURL:   baseURL,
    }
}

func (s *URLService) ShortenURL(originalURL string) (string, error) {
    shortID := s.generator.Generate()
    if err := s.storage.Save(shortID, originalURL); err != nil {
        return "", err
    }
    return s.baseURL + "/" + shortID, nil
}

func (s *URLService) GetOriginalURL(shortID string) (string, bool) {
    return s.storage.Get(shortID)
}