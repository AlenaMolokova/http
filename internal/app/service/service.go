package service

import (
	"context"
	"fmt"

	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
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
	}
}

func (s *Service) ShortenURL(ctx context.Context, originalURL, userID string) (models.ShortenResult, error) {
	logrus.WithFields(logrus.Fields{
        "originalURL": originalURL,
        "userID":      userID,
    }).Debug("Shortening URL")
    
    existingShortID, err := s.saver.FindByOriginalURL(ctx, originalURL)
    if err != nil {
        logrus.WithError(err).Error("Error finding URL")
        return models.ShortenResult{}, fmt.Errorf("error finding URL: %w", err)
    }
    if existingShortID != "" {
        logrus.WithField("shortID", existingShortID).Info("URL already exists")
        return models.ShortenResult{
            ShortURL: fmt.Sprintf("%s/%s", s.BaseURL, existingShortID),
            IsNew:    false,
        }, nil
    }

    shortID := s.generator.Generate()
    if shortID == "" {
        logrus.Error("Generated short ID is empty")
        return models.ShortenResult{}, fmt.Errorf("failed to generate short ID")
    }

    if err := s.saver.Save(ctx, shortID, originalURL, userID); err != nil {
        logrus.WithError(err).Error("Error saving URL")
        return models.ShortenResult{}, fmt.Errorf("error saving URL: %w", err)
    }

    logrus.WithField("shortID", shortID).Info("URL shortened successfully")
    return models.ShortenResult{
        ShortURL: fmt.Sprintf("%s/%s", s.BaseURL, shortID),
        IsNew:    true,
    }, nil
}

func (s *Service) ShortenBatch(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
	batch := make(map[string]string)
	for _, item := range items {
		shortID := s.generator.Generate()
		batch[shortID] = item.OriginalURL
	}

	if err := s.batch.SaveBatch(ctx, batch, userID); err != nil {
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

func (s *Service) Get(ctx context.Context, shortID string) (string, bool) {
	return s.getter.Get(ctx, shortID)
}

func (s *Service) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	urls, err := s.fetcher.GetURLsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения URL пользователя: %w", err)
	}
	for i := range urls {
		urls[i].ShortURL = fmt.Sprintf("%s/%s", s.BaseURL, urls[i].ShortURL)
	}
	return urls, nil
}

func (s *Service) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	err := s.deleter.DeleteURLs(ctx, shortIDs, userID)
    if err != nil {
        logrus.WithError(err).Error("Failed to delete URLs")
        return err
    }
    return nil
}

func (s *Service) Ping(ctx context.Context) error {
	return s.pinger.Ping(ctx)
}