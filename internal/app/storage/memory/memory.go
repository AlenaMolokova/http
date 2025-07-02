// Package memory предоставляет реализацию хранилища URL в оперативной памяти.
package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/AlenaMolokova/http/internal/app/models"
)

// MemoryStorage предоставляет хранилище URL-адресов в оперативной памяти.
// Данные хранятся только во время работы программы и теряются при её завершении.
// Поддерживает конкурентный доступ через механизмы синхронизации.
type MemoryStorage struct {
	urls map[string]models.UserURL
	mu   sync.RWMutex
}

// NewMemoryStorage создаёт и инициализирует новое хранилище URL-адресов в памяти.
//
// Возвращает:
//   - указатель на MemoryStorage с пустым хранилищем URL-адресов
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls: make(map[string]models.UserURL),
	}
}

// Save сохраняет новый URL-адрес в хранилище.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращенный идентификатор URL
//   - originalURL: оригинальный URL-адрес
//   - userID: идентификатор пользователя, который создал сокращение
//
// Возвращает:
//   - ошибку, если не удалось сохранить URL (в текущей реализации всегда nil)
func (s *MemoryStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[shortID] = models.UserURL{
		ShortURL:    shortID,
		OriginalURL: originalURL,
		UserID:      userID,
		IsDeleted:   false,
	}
	return nil
}

// FindByOriginalURL ищет сокращенный идентификатор по оригинальному URL-адресу.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - originalURL: оригинальный URL-адрес для поиска
//
// Возвращает:
//   - сокращенный идентификатор, если URL найден и не помечен как удаленный
//   - пустую строку, если URL не найден
//   - ошибку (в текущей реализации всегда nil)
func (s *MemoryStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for shortID, url := range s.urls {
		if url.OriginalURL == originalURL && !url.IsDeleted {
			return shortID, nil
		}
	}
	return "", nil
}

// SaveBatch сохраняет пакет URL-адресов в хранилище.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - items: карта, где ключ - сокращенный идентификатор, значение - оригинальный URL
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - ошибку, если не удалось сохранить пакет URL-адресов (в текущей реализации всегда nil)
func (s *MemoryStorage) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for shortID, originalURL := range items {
		s.urls[shortID] = models.UserURL{
			ShortURL:    shortID,
			OriginalURL: originalURL,
			UserID:      userID,
			IsDeleted:   false,
		}
	}
	return nil
}

// Get возвращает оригинальный URL-адрес по сокращенному идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращенный идентификатор URL
//
// Возвращает:
//   - оригинальный URL-адрес и true, если сокращение найдено и не удалено
//   - пустую строку и false, если сокращение не найдено или удалено
func (s *MemoryStorage) Get(ctx context.Context, shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, exists := s.urls[shortID]
	if !exists || url.IsDeleted {
		return "", false
	}
	return url.OriginalURL, true
}

// GetURLsByUserID возвращает все неудаленные URL-адреса, созданные указанным пользователем.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - список структур UserURL, содержащих сокращенные и оригинальные URL-адреса
//   - ошибку (в текущей реализации всегда nil)
func (s *MemoryStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.UserURL
	for _, url := range s.urls {
		if url.UserID == userID && !url.IsDeleted {
			result = append(result, url)
		}
	}
	return result, nil
}

// DeleteURLs помечает указанные URL-адреса как удаленные.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortIDs: список сокращенных идентификаторов для удаления
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - ошибку, если не удалось пометить URL-адреса как удаленные (в текущей реализации всегда nil)
func (s *MemoryStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, shortID := range shortIDs {
		if url, exists := s.urls[shortID]; exists && url.UserID == userID {
			url.IsDeleted = true
			s.urls[shortID] = url
		}
	}
	return nil
}

// Ping проверяет доступность хранилища.
// Поскольку это хранилище в памяти, метод всегда возвращает ошибку,
// указывающую на то, что проверка соединения не поддерживается.
//
// Параметры:
//   - ctx: контекст выполнения операции
//
// Возвращает:
//   - ошибку с сообщением о неподдерживаемой операции
func (s *MemoryStorage) Ping(ctx context.Context) error {
	return errors.New("memory storage does not support database connection check")
}

// GetStats возвращает статистику сервиса: количество сокращенных URL и пользователей.
//
// Параметры:
//   - ctx: контекст выполнения операции
//
// Возвращает:
//   - структуру Stats с информацией о количестве URL и пользователей
//   - ошибку (в текущей реализации всегда nil)
func (s *MemoryStorage) GetStats(ctx context.Context) (models.Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var urlsCount int
	userIDs := make(map[string]bool)

	for _, url := range s.urls {
		if !url.IsDeleted {
			urlsCount++
			userIDs[url.UserID] = true
		}
	}

	return models.Stats{
		URLs:  urlsCount,
		Users: len(userIDs),
	}, nil
}

// ShortenURL создает сокращенную версию оригинального URL для указанного пользователя.
// Реализует интерфейс URLShortener.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - originalURL: оригинальный URL для сокращения
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - структуру ShortenResult с сокращенным URL и флагом новизны
//   - ошибку при неудачном выполнении операции
func (s *MemoryStorage) ShortenURL(ctx context.Context, originalURL, userID string) (models.ShortenResult, error) {
	// Проверяем, существует ли уже такой URL
	if shortID, err := s.FindByOriginalURL(ctx, originalURL); err == nil && shortID != "" {
		return models.ShortenResult{
			ShortURL: shortID,
			IsNew:    false,
		}, nil
	}

	// Генерируем новый короткий идентификатор (простая заглушка)
	shortID := generateShortID()

	// Сохраняем новый URL
	if err := s.Save(ctx, shortID, originalURL, userID); err != nil {
		return models.ShortenResult{}, err
	}

	return models.ShortenResult{
		ShortURL: shortID,
		IsNew:    true,
	}, nil
}

// ShortenBatch выполняет пакетное сокращение URL для заданного списка запросов и пользователя.
// Реализует интерфейс BatchURLShortener.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - items: список запросов на сокращение URL
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - список ответов с сокращенными URL
//   - ошибку при неудачном выполнении операции
func (s *MemoryStorage) ShortenBatch(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
	var responses []models.BatchShortenResponse
	batchItems := make(map[string]string)

	for _, item := range items {
		// Проверяем, существует ли уже такой URL
		if shortID, err := s.FindByOriginalURL(ctx, item.OriginalURL); err == nil && shortID != "" {
			responses = append(responses, models.BatchShortenResponse{
				CorrelationID: item.CorrelationID,
				ShortURL:      shortID,
			})
		} else {
			// Генерируем новый короткий идентификатор
			shortID := generateShortID()
			batchItems[shortID] = item.OriginalURL
			responses = append(responses, models.BatchShortenResponse{
				CorrelationID: item.CorrelationID,
				ShortURL:      shortID,
			})
		}
	}

	// Сохраняем новые URL пачкой
	if len(batchItems) > 0 {
		if err := s.SaveBatch(ctx, batchItems, userID); err != nil {
			return nil, err
		}
	}

	return responses, nil
}

// generateShortID генерирует короткий идентификатор для URL.
// Это упрощенная реализация для демонстрации.
func generateShortID() string {
	// В реальной реализации здесь должен быть более сложный алгоритм
	// генерации уникальных коротких идентификаторов
	return "short_id_placeholder"
}
