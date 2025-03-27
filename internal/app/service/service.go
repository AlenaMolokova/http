package service

import (
	"fmt"

	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/models"
)

type ShortenResult struct {
    ShortURL string
    IsNew    bool
}

type URLStorage interface {
	Save(shortID, originalURL, userID string) error
	SaveBatch(items map[string]string, userID string) error
	Get(shortID string) (string, bool)
	FindByOriginalURL(originalURL string) (string, error)
	GetURLsByUserID(userID string) ([]models.UserURL, error)
	Ping() error
}

type URLService interface {
	ShortenURL(originalURL, userID string) (ShortenResult, error)
	ShortenBatch(items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error)
	GetOriginalURL(shortID string) (string, bool)
	GetURLsByUserID(userID string) ([]models.UserURL, error)
	Ping() error
	GetUserURLs(userID string) ([]models.UserURL, error)
	FindByOriginalURL(originalURL string) (string, error)
}

type service struct {
	storage   URLStorage
	generator generator.Generator
	baseURL   string
}

func (s *service) ShortenURL(originalURL, userID string) (ShortenResult, error) {
	existingShortID, err := s.storage.FindByOriginalURL(originalURL)
	if err == nil && existingShortID != "" {
		return ShortenResult{
			ShortURL: fmt.Sprintf("%s/%s", s.baseURL, existingShortID),
			IsNew:    false,
		}, nil
	}

	shortID := s.generator.Generate()
	if err := s.storage.Save(shortID, originalURL, userID); err != nil {
		return ShortenResult{}, fmt.Errorf("ошибка сохранения URL: %w", err)
	}
	return ShortenResult{
		ShortURL: fmt.Sprintf("%s/%s", s.baseURL, shortID),
		IsNew:    true,
	}, nil
}

func NewURLService(storage URLStorage, generator generator.Generator, baseURL string) URLService {
	return &service{
		storage:   storage,
		generator: generator,
		baseURL:   baseURL,
	}
}

func (s *service) GetOriginalURL(shortID string) (string, bool) {
	return s.storage.Get(shortID)
}

func (s *service) Ping() error {
	return s.storage.Ping()
}

func (s *service) ShortenBatch(items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
	var result []models.BatchShortenResponse
	batchMap := make(map[string]string)
	correlationMap := make(map[string]string)

	for _, item := range items {
		shortID := s.generator.Generate()
		batchMap[shortID] = item.OriginalURL
		correlationMap[shortID] = item.CorrelationID
	}

	if err := s.storage.SaveBatch(batchMap, userID); err != nil {
		return nil, fmt.Errorf("ошибка пакетного сохранения URL: %w", err)
	}

	for shortID, correlationID := range correlationMap {
		result = append(result, models.BatchShortenResponse{
			CorrelationID: correlationID,
			ShortURL:      fmt.Sprintf("%s/%s", s.baseURL, shortID),
		})
	}
	return result, nil
}

func (s *service) GetUserURLs(userID string) ([]models.UserURL, error) {
	return s.storage.GetURLsByUserID(userID)
}

func (s *service) GetURLsByUserID(userID string) ([]models.UserURL, error) {
	urls, err := s.storage.GetURLsByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения URL пользователя: %w", err)
	}

	for i := range urls {
		urls[i].ShortURL = fmt.Sprintf("%s/%s", s.baseURL, urls[i].ShortURL)
	}

	return urls, nil
}

func (s *service) FindByOriginalURL(originalURL string) (string, error) {
	return s.storage.FindByOriginalURL(originalURL)
}
