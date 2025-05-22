package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/models"
)

// Service представляет собой слой бизнес-логики для сервиса сокращения URL.
// Он обрабатывает сокращение URL, получение оригинальных URL по сокращенным идентификаторам,
// управление пакетными операциями с URL и кэширование данных пользователя.
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

// NewService создает и инициализирует новый экземпляр сервиса с предоставленными зависимостями.
//
// Параметры:
//   - saver: интерфейс для сохранения URL
//   - batch: интерфейс для пакетного сохранения URL
//   - getter: интерфейс для получения оригинальных URL по их коротким идентификаторам
//   - fetcher: интерфейс для получения всех URL, связанных с конкретным пользователем
//   - deleter: интерфейс для удаления URL
//   - pinger: интерфейс для проверки соединения с хранилищем
//   - generator: генератор коротких идентификаторов
//   - baseURL: базовый URL сервиса, используемый для создания полных сокращенных URL
//
// Возвращает:
//   - *Service: указатель на новый экземпляр сервиса
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

// ShortenURL сокращает оригинальный URL, создавая для него короткий идентификатор.
// Если URL уже был сокращен ранее, возвращает существующий короткий URL.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - originalURL: оригинальный URL для сокращения
//   - userID: идентификатор пользователя, запрашивающего сокращение
//
// Возвращает:
//   - models.ShortenResult: результат операции сокращения, включающий короткий URL и флаг новизны
//   - error: ошибка, если операция не удалась
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

// ShortenBatch выполняет пакетное сокращение нескольких URL одновременно.
// Для каждого URL в пакете создается уникальный короткий идентификатор.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - items: список запросов на сокращение с корреляционными идентификаторами
//   - userID: идентификатор пользователя, запрашивающего сокращение
//
// Возвращает:
//   - []models.BatchShortenResponse: список результатов пакетного сокращения
//   - error: ошибка, если операция не удалась
func (s *Service) ShortenBatch(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
	batch := make(map[string]string, len(items))

	correlationMap := make(map[string]string, len(items)) // correlationID -> shortID

	for _, item := range items {
		shortID := s.generator.Generate()
		batch[shortID] = item.OriginalURL
		correlationMap[item.CorrelationID] = shortID
	}

	s.cacheMu.Lock()
	delete(s.cache, userID)
	s.cacheMu.Unlock()

	if err := s.batch.SaveBatch(ctx, batch, userID); err != nil {
		return nil, fmt.Errorf("ошибка сохранения пакета URL: %w", err)
	}

	resp := make([]models.BatchShortenResponse, 0, len(items))
	for _, item := range items {
		shortID := correlationMap[item.CorrelationID]
		resp = append(resp, models.BatchShortenResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", s.BaseURL, shortID),
		})
	}

	return resp, nil
}

// Get возвращает оригинальный URL по его короткому идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: короткий идентификатор URL
//
// Возвращает:
//   - string: оригинальный URL
//   - bool: флаг успешности операции (true, если URL найден)
func (s *Service) Get(ctx context.Context, shortID string) (string, bool) {
	return s.getter.Get(ctx, shortID)
}

// GetURLsByUserID возвращает все URL, созданные конкретным пользователем.
// Результаты кэшируются для повышения производительности последующих запросов.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - []models.UserURL: список URL пользователя
//   - error: ошибка, если операция не удалась
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

// DeleteURLs удаляет указанные URL, принадлежащие конкретному пользователю.
// После удаления кэш пользователя очищается.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortIDs: список коротких идентификаторов URL для удаления
//   - userID: идентификатор пользователя, которому принадлежат URL
//
// Возвращает:
//   - error: ошибка, если операция не удалась
func (s *Service) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	s.cacheMu.Lock()
	delete(s.cache, userID)
	s.cacheMu.Unlock()

	return s.deleter.DeleteURLs(ctx, shortIDs, userID)
}

// Ping проверяет соединение с хранилищем данных.
//
// Параметры:
//   - ctx: контекст выполнения операции
//
// Возвращает:
//   - error: ошибка, если проверка соединения не удалась
func (s *Service) Ping(ctx context.Context) error {
	return s.pinger.Ping(ctx)
}
