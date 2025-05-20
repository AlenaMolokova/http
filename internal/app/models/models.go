package models

import (
	"context"
	"encoding/json"
)

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
}

type URLWithUser struct {
	ShortID     string
	OriginalURL string
	UserID      string
}

type ShortenResult struct {
	ShortURL string `json:"short_url"`
	IsNew    bool   `json:"is_new"`
}

type URLShortener interface {
	ShortenURL(ctx context.Context, originalURL, userID string) (ShortenResult, error)
}

type BatchURLShortener interface {
	ShortenBatch(ctx context.Context, items []BatchShortenRequest, userID string) ([]BatchShortenResponse, error)
}

type URLGetter interface {
	Get(ctx context.Context, shortID string) (string, bool)
}

type URLFetcher interface {
	GetURLsByUserID(ctx context.Context, userID string) ([]UserURL, error)
}

type URLDeleter interface {
	DeleteURLs(ctx context.Context, shortIDs []string, userID string) error
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type URLSaver interface {
	Save(ctx context.Context, shortID, originalURL, userID string) error
	FindByOriginalURL(ctx context.Context, originalURL string) (string, error)
}

type URLBatchSaver interface {
	SaveBatch(ctx context.Context, items map[string]string, userID string) error
}

func (r ShortenResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Result string `json:"result"`
	}{
		Result: r.Result,
	})
}

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
