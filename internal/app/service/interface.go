package service

type URLService interface {
	ShortenURL(originalURL string) (string, error)
	GetOriginalURL(shortID string) (string, bool)
}
