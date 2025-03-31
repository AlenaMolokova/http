package service

import (
	"context"
	"fmt"

	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
)

// Интерфейсы
type URLSaver interface {
	Save(ctx context.Context, shortID, originalURL, userID string) error
	FindByOriginalURL(ctx context.Context, originalURL string) (string, error)
}

type URLBatchSaver interface {
	SaveBatch(ctx context.Context, items map[string]string, userID string) error
}

type URLGetter interface {
	Get(ctx context.Context, shortID string) (string, bool)
}

type URLFetcher interface {
	GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error)
}

type URLDeleter interface {
	DeleteURLs(ctx context.Context, shortIDs []string, userID string) error
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type Shortener interface {
	ShortenURL(ctx context.Context, originalURL, userID string) (ShortenResult, error)
	ShortenBatch(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error)
}

type ShortenResult struct {
	ShortURL string
	IsNew    bool
}

type service struct {
	saver      URLSaver
	batchSaver URLBatchSaver
	getter     URLGetter
	fetcher    URLFetcher
	deleter    URLDeleter
	pinger     Pinger
	generator  generator.Generator
	BaseURL    string
}

func NewService(saver URLSaver, batchSaver URLBatchSaver, getter URLGetter, fetcher URLFetcher, deleter URLDeleter, pinger Pinger, generator generator.Generator, baseURL string) *service {
	return &service{
		saver:      saver,
		batchSaver: batchSaver,
		getter:     getter,
		fetcher:    fetcher,
		deleter:    deleter,
		pinger:     pinger,
		generator:  generator,
		BaseURL:    baseURL,
	}
}

func (s *service) ShortenURL(ctx context.Context, originalURL, userID string) (ShortenResult, error) {
	existingShortID, err := s.saver.FindByOriginalURL(ctx, originalURL)
	if err == nil && existingShortID != "" {
		return ShortenResult{ShortURL: fmt.Sprintf("%s/%s", s.BaseURL, existingShortID), IsNew: false}, nil
	}

	shortID := s.generator.Generate()
	if err := s.saver.Save(ctx, shortID, originalURL, userID); err != nil {
		return ShortenResult{}, fmt.Errorf("ошибка сохранения URL: %w", err)
	}
	return ShortenResult{ShortURL: fmt.Sprintf("%s/%s", s.BaseURL, shortID), IsNew: true}, nil
}

func (s *service) ShortenBatch(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
	batch := make(map[string]string)
	for _, item := range items {
		batch[s.generator.Generate()] = item.OriginalURL
	}
	if err := s.batchSaver.SaveBatch(ctx, batch, userID); err != nil {
		return nil, fmt.Errorf("ошибка сохранения пакета URL: %w", err)
	}

	var resp []models.BatchShortenResponse
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

func (s *service) Get(ctx context.Context, shortID string) (string, bool) {
	return s.getter.Get(ctx, shortID)
}

func (s *service) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	urls, err := s.fetcher.GetURLsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения URL пользователя: %w", err)
	}
	for i := range urls {
		urls[i].ShortURL = fmt.Sprintf("%s/%s", s.BaseURL, urls[i].ShortURL)
	}
	return urls, nil
}

func (s *service) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
    workers := make(chan struct{}, 4)
    results := make(chan error, len(shortIDs))
    done := make(chan struct{})

    go func() {
        defer close(done)
        for _, shortID := range shortIDs {
            workers <- struct{}{}
            go func(id string) {
                defer func() { <-workers }()
                err := s.deleter.DeleteURLs(context.Background(), []string{id}, userID)
                results <- err
            }(shortID)
        }

        for range shortIDs {
            if err := <-results; err != nil {
                logrus.WithError(err).Error("Failed to delete URL in background")
            }
        }
    }()

    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (s *service) Ping(ctx context.Context) error {
	return s.pinger.Ping(ctx)
}