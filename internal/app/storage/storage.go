package storage

import (
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/internal/app/storage/database"
	"github.com/AlenaMolokova/http/internal/app/storage/file"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	impl interface{}
}

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

func (s *Storage) AsURLSaver() models.URLSaver {
	return s.impl.(models.URLSaver)
}

func (s *Storage) AsURLBatchSaver() models.URLBatchSaver {
	return s.impl.(models.URLBatchSaver)
}

func (s *Storage) AsURLGetter() models.URLGetter {
	return s.impl.(models.URLGetter)
}

func (s *Storage) AsURLFetcher() models.URLFetcher {
	return s.impl.(models.URLFetcher)
}

func (s *Storage) AsURLDeleter() models.URLDeleter {
	return s.impl.(models.URLDeleter)
}

func (s *Storage) AsPinger() models.Pinger {
	return s.impl.(models.Pinger)
}
