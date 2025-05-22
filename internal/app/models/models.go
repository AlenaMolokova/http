package models

import (
	"context"
	"encoding/json"
)

// ShortenRequest представляет запрос на сокращение URL.
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse представляет ответ на запрос о сокращении URL.
type ShortenResponse struct {
	Result string `json:"result"`
}

// BatchShortenRequest представляет элемент запроса для пакетного сокращения URL.
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse представляет элемент ответа для пакетного сокращения URL.
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURL представляет информацию о сокращенном URL, связанном с пользователем.
type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
}

// URLWithUser представляет информацию о сокращенном URL с идентификатором пользователя.
type URLWithUser struct {
	ShortID     string
	OriginalURL string
	UserID      string
}

// ShortenResult представляет результат операции сокращения URL.
type ShortenResult struct {
	ShortURL string `json:"short_url"`
	IsNew    bool   `json:"is_new"`
}

// URLShortener интерфейс, определяющий методы для сокращения URL.
type URLShortener interface {
	// ShortenURL создает сокращенную версию оригинального URL для указанного пользователя.
	// Возвращает структуру ShortenResult, содержащую короткий URL и флаг, указывающий,
	// был ли URL создан впервые или уже существовал.
	ShortenURL(ctx context.Context, originalURL, userID string) (ShortenResult, error)
}

// BatchURLShortener интерфейс, определяющий методы для пакетного сокращения URL.
type BatchURLShortener interface {
	// ShortenBatch выполняет пакетное сокращение URL для заданного списка запросов и пользователя.
	// Возвращает список ответов с сокращенными URL для каждого запроса.
	ShortenBatch(ctx context.Context, items []BatchShortenRequest, userID string) ([]BatchShortenResponse, error)
}

// URLGetter интерфейс, определяющий методы для получения оригинального URL по короткому идентификатору.
type URLGetter interface {
	// Get возвращает оригинальный URL по короткому идентификатору.
	// Второй возвращаемый параметр указывает, был ли найден URL.
	Get(ctx context.Context, shortID string) (string, bool)
}

// URLFetcher интерфейс, определяющий методы для получения URL пользователя.
type URLFetcher interface {
	// GetURLsByUserID возвращает список URL, связанных с указанным пользователем.
	GetURLsByUserID(ctx context.Context, userID string) ([]UserURL, error)
}

// URLDeleter интерфейс, определяющий методы для удаления URL.
type URLDeleter interface {
	// DeleteURLs удаляет указанные URL для заданного пользователя.
	DeleteURLs(ctx context.Context, shortIDs []string, userID string) error
}

// Pinger интерфейс, определяющий методы для проверки доступности сервиса.
type Pinger interface {
	// Ping проверяет доступность сервиса.
	Ping(ctx context.Context) error
}

// URLSaver интерфейс, определяющий методы для сохранения URL.
type URLSaver interface {
	// Save сохраняет короткий идентификатор, оригинальный URL и идентификатор пользователя.
	Save(ctx context.Context, shortID, originalURL, userID string) error

	// FindByOriginalURL ищет короткий идентификатор по оригинальному URL.
	FindByOriginalURL(ctx context.Context, originalURL string) (string, error)
}

// URLBatchSaver интерфейс, определяющий методы для пакетного сохранения URL.
type URLBatchSaver interface {
	// SaveBatch сохраняет пакет URL для указанного пользователя.
	SaveBatch(ctx context.Context, items map[string]string, userID string) error
}

// MarshalJSON реализует интерфейс json.Marshaler для типа ShortenResponse.
// Преобразует структуру ShortenResponse в формат JSON.
func (r ShortenResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Result string `json:"result"`
	}{
		Result: r.Result,
	})
}

// UnmarshalJSON реализует интерфейс json.Unmarshaler для типа ShortenRequest.
// Преобразует данные в формате JSON в структуру ShortenRequest.
func (r *ShortenRequest) UnmarshalJSON(data []byte) error {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		return err
	}
	r.URL = req.URL
	return nil
}
