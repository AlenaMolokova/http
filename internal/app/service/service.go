package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/models"
)

type Service struct {
	saver     models.URLSaver
	batch     models.URLBatchSaver
	getter    models.URLGetter
	fetcher   models.URLFetcher
	deleter   models.URLDeleter
	pinger    models.Pinger
	generator generator.Generator
	BaseURL   string
	cache     map[string][]models.UserURL
	cacheMu   sync.RWMutex
}

func NewService(saver models.URLSaver, batch models.URLBatchSaver, getter models.URLGetter, fetcher models.URLFetcher, deleter models.URLDeleter, pinger models.Pinger, generator generator.Generator, baseURL string) *Service {
	return &Service{
		saver:     saver,
		batch:     batch,
		getter:    getter,
		fetcher:   fetcher,
		deleter:   deleter,
		pinger:    pinger,
		generator: generator,
		BaseURL:   baseURL,
		cache:     make(map[string][]models.UserURL),
	}
}

func (s *Service) ShortenURL(ctx context.Context, originalURL, userID string) (models.ShortenResult, error) {
	existingShortID, err := s.saver.FindByOriginalURL(ctx, originalURL)
	if err != nil {
		return models.ShortenResult{}, fmt.Errorf("error finding URL: %w", err)
	}
	if existingShortID != "" {
		return models.ShortenResult{
			ShortURL: fmt.Sprintf("%s/%s", s.BaseURL, existingShortID),
			IsNew:    false,
		}, nil
	}

	shortID := s.generator.Generate()
	if shortID == "" {
		return models.ShortenResult{}, fmt.Errorf("failed to generate short ID")
	}

	s.cacheMu.Lock()
	delete(s.cache, userID)
	s.cacheMu.Unlock()

	if err := s.saver.Save(ctx, shortID, originalURL, userID); err != nil {
		return models.ShortenResult{}, fmt.Errorf("error saving URL: %w", err)
	}

	return models.ShortenResult{
		ShortURL: fmt.Sprintf("%s/%s", s.BaseURL, shortID),
		IsNew:    true,
	}, nil
}

func (s *Service) ShortenBatch(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
	batch := make(map[string]string, len(items))
	for _, item := range items {
		shortID := s.generator.Generate()
		batch[shortID] = item.OriginalURL
	}

	s.cacheMu.Lock()
	delete(s.cache, userID)
	s.cacheMu.Unlock()

	if err := s.batch.SaveBatch(ctx, batch, userID); err != nil {
		return nil, fmt.Errorf("ошибка сохранения пакета URL: %w", err)
	}

	resp := make([]models.BatchShortenResponse, 0, len(items))
	for shortID, originalURL := range batch {
		for _, item := range items {
			if item.OriginalURL == originalURL {
				resp = append(resp, models.BatchShortenResponse{
					CorrelationID: item.CorrelationID,
					ShortURL:      fmt.Sprintf("%s/%s", s.BaseURL, shortID),
				})
				break
			}
		}
	}
	return resp, nil
}

func (s *Service) Get(ctx context.Context, shortID string) (string, bool) {
	return s.getter.Get(ctx, shortID)
}

func (s *Service) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	s.cacheMu.RLock()
	cached, ok := s.cache[userID]
	s.cacheMu.RUnlock()
	if ok {
		return cached, nil
	}

	urls, err := s.fetcher.GetURLsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения URL пользователя: %w", err)
	}

	for i := range urls {
		urls[i].ShortURL = fmt.Sprintf("%s/%s", s.BaseURL, urls[i].ShortURL)
	}

	s.cacheMu.Lock()
	s.cache[userID] = urls
	s.cacheMu.Unlock()

	return urls, nil
}

func (s *Service) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	s.cacheMu.Lock()
	delete(s.cache, userID)
	s.cacheMu.Unlock()

	return s.deleter.DeleteURLs(ctx, shortIDs, userID)
}

func (s *Service) Ping(ctx context.Context) error {
	return s.pinger.Ping(ctx)
}
