package storage

import (
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/internal/app/storage/database"
	"github.com/AlenaMolokova/http/internal/app/storage/file"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/sirupsen/logrus"
)

// Storage представляет собой обертку для различных реализаций хранилища.
// Он скрывает конкретную реализацию хранилища и предоставляет унифицированный интерфейс доступа.
type Storage struct {
	impl interface{}
}

// NewStorage creates a new storage instance based on provided parameters.
// The function tries to use storages in the following priority order:
// 1. PostgreSQL (if databaseDSN is not empty)
// 2. File storage (if fileStoragePath is not empty)
// 3. In-memory storage (default)
//
// Parameters:
//   - databaseDSN: PostgreSQL database connection string
//   - fileStoragePath: path to file for data storage
//
// Returns:
//   - *Storage: pointer to a new storage instance
//   - error: error if storage initialization fails
func NewStorage(databaseDSN, fileStoragePath string) (*Storage, error) {
	var impl interface{}

	if databaseDSN != "" {
		dbStorage, err := database.NewPostgresStorage(databaseDSN)
		if err == nil {
			logrus.Info("Используется хранилище PostgreSQL")
			impl = dbStorage
		} else {
			logrus.WithError(err).Warn("Не удалось использовать PostgreSQL, переходим к следующему варианту")
		}
	}

	if impl == nil && fileStoragePath != "" {
		fileStorage, err := file.NewFileStorage(fileStoragePath)
		if err == nil {
			logrus.WithField("file", fileStoragePath).Info("Используется файловое хранилище")
			impl = fileStorage
		} else {
			logrus.WithError(err).Warn("Не удалось использовать файловое хранилище, переходим к памяти")
		}
	}

	if impl == nil {
		logrus.Info("Используется хранилище в памяти")
		impl = memory.NewMemoryStorage()
	}

	return &Storage{impl: impl}, nil
}

// AsURLSaver returns URLSaver interface implementation for the current storage.
//
// Returns:
//   - models.URLSaver: interface for URL saving
func (s *Storage) AsURLSaver() models.URLSaver {
	return s.impl.(models.URLSaver)
}

// AsURLBatchSaver returns URLBatchSaver interface implementation for the current storage.
//
// Returns:
//   - models.URLBatchSaver: interface for batch URL saving
func (s *Storage) AsURLBatchSaver() models.URLBatchSaver {
	return s.impl.(models.URLBatchSaver)
}

// AsBatchURLShortener returns BatchURLShortener interface implementation for the current storage.
//
// Returns:
//   - models.BatchURLShortener: interface for batch URL shortening
func (s *Storage) AsBatchURLShortener() models.BatchURLShortener {
	return s.impl.(models.BatchURLShortener)
}

// AsURLGetter returns URLGetter interface implementation for the current storage.
//
// Returns:
//   - models.URLGetter: interface for getting URLs by short identifier
func (s *Storage) AsURLGetter() models.URLGetter {
	return s.impl.(models.URLGetter)
}

// AsURLFetcher returns URLFetcher interface implementation for the current storage.
//
// Returns:
//   - models.URLFetcher: interface for fetching URLs by user identifier
func (s *Storage) AsURLFetcher() models.URLFetcher {
	return s.impl.(models.URLFetcher)
}

// AsURLDeleter returns URLDeleter interface implementation for the current storage.
//
// Returns:
//   - models.URLDeleter: interface for deleting URLs
func (s *Storage) AsURLDeleter() models.URLDeleter {
	return s.impl.(models.URLDeleter)
}

// AsPinger returns Pinger interface implementation for the current storage.
//
// Returns:
//   - models.Pinger: interface for checking connection to storage
func (s *Storage) AsPinger() models.Pinger {
	return s.impl.(models.Pinger)
}
