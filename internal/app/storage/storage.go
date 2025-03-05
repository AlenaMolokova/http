package storage

import (
	"github.com/AlenaMolokova/http/internal/app/storage/database"
	"github.com/AlenaMolokova/http/internal/app/storage/file"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/sirupsen/logrus"
)

func InitStorage(databaseDSN, fileStoragePath string) (URLStorage, error) {
	
	if databaseDSN != "" {
		dbStorage, err := database.NewDBStorage(databaseDSN)
		if err != nil {
			logrus.WithError(err).Error("Не удалось инициализировать хранилище в БД")
		} else {
			logrus.Info("Используется хранилище в базе данных PostgreSQL")
			return dbStorage, nil
		}
	}

	if fileStoragePath != "" {
		fileStorage, err := file.NewFileStorage(fileStoragePath)
		if err != nil {
			logrus.WithError(err).Error("Не удалось инициализировать файловое хранилище")
		} else {
			logrus.WithField("file", fileStoragePath).Info("Используется файловое хранилище")
			return fileStorage, nil
		}
	}

	logrus.Info("Используется хранилище в памяти")
	return memory.NewMemoryStorage(), nil
}

type URLStorage interface {
	Save(shortID, originalURL string) error
	Get(shortID string) (string, bool)
	Ping() error
}