package service

import (
	"errors"
    "github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/models"
)

type URLStorage interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, bool)
	FindByOriginalURL(originalURL string) (string, error)
	Ping() error
}

type URLService interface {
	ShortenURL(originalURL string) (string, error)
	ShortenBatch(items []models.BatchShortenRequest) ([]models.BatchShortenResponse, error)
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
	shortID, err := s.storage.FindByOriginalURL(originalURL)
	if err ==nil {
		return s.baseURL + "/" + shortID, errors.New("url already exists")
	}

	shortID = s.generator.Generate()
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

func (s *service) ShortenBatch(items []models.BatchShortenRequest) ([]models.BatchShortenResponse, error) {
    var result []models.BatchShortenResponse
    
    for _, item := range items {
        shortID := s.generator.Generate()
        if err := s.storage.Save(shortID, item.OriginalURL); err != nil {
            return nil, err
        }
        
        result = append(result, models.BatchShortenResponse{
            CorrelationID: item.CorrelationID,
            ShortURL:      s.baseURL + "/" + shortID,
        })
    }
    
    return result, nil
}