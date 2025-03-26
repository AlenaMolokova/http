package storage

import (
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/internal/app/storage/database"
	"github.com/AlenaMolokova/http/internal/app/storage/file"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/sirupsen/logrus"
)

func NewStorage(databaseDSN, fileStoragePath string) (URLStorage, error) {
	
	if databaseDSN != "" {
		dbStorage, err := database.NewPostgresStorage(databaseDSN)
		if err == nil {
			logrus.Info("Используется хранилище PostgreSQL")
			return dbStorage, nil
		}
		logrus.WithError(err).Warn("Не удалось использовать PostgreSQL")
	}

	if fileStoragePath != "" {
		fileStorage, err := file.NewFileStorage(fileStoragePath)
		if err == nil {
			logrus.WithField("file", fileStoragePath).Info("Используется файловое хранилище")
			return fileStorage, nil
		}
		logrus.WithError(err).Warn("Не удалось использовать файловое хранилище")
	}

	logrus.Info("Используется хранилище в памяти")
	return memory.NewMemoryStorage(), nil
}

type URLStorage interface {
	Save(shortID, originalURL, userID string) error
	SaveBatch(items map[string]string, userID string) error
	Get(shortID string) (string, bool)
	FindByOriginalURL(originalURL string) (string, error)
	GetURLsByUserID(userID string) ([]models.UserURL, error)
	Ping() error
}