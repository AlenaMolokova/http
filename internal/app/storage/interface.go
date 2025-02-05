package storage

type URLStorage interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, bool)
}
