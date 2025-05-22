package app

import (
	"context"
	"time"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage"
	"github.com/sirupsen/logrus"
)

// App представляет собой основную структуру приложения,
// содержащую обработчик URL и сервисный слой.
type App struct {
	Handler *handler.URLHandler
	Service *service.Service
}

// GenerateTestLoad генерирует тестовую нагрузку, создавая указанное
// количество сокращенных URL для тестового пользователя.
//
// Параметры:
//   - count: количество URL для генерации
//
// Метод также выполняет операции чтения, чтобы проверить
// корректность сохранения и получения данных.
func (a *App) GenerateTestLoad(count int) {
	ctx := context.Background()
	userID := "test-user"

	logrus.Info("Generating test load: ", count, " URLs")

	for i := 0; i < count; i++ {
		originalURL := "https://example.com/" + time.Now().String() + "/" + generator.NewGenerator(4).Generate()
		_, err := a.Service.ShortenURL(ctx, originalURL, userID)
		if err != nil {
			logrus.WithError(err).Warn("Failed to shorten URL during test load")
		}

		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	urls, err := a.Service.GetURLsByUserID(ctx, userID)
	if err != nil {
		logrus.WithError(err).Warn("Failed to get user URLs during test load")
	} else {
		logrus.Info("Retrieved ", len(urls), " URLs for test user")
	}

	if len(urls) > 0 {
		for i := 0; i < min(10, len(urls)); i++ {
			shortID := urls[i].ShortURL
			if len(shortID) > 8 {
				shortID = shortID[len(shortID)-8:]
				_, found := a.Service.Get(ctx, shortID)
				if !found {
					logrus.Warn("URL not found during test load: ", shortID)
				}
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewApp создает и инициализирует новый экземпляр приложения.
//
// Параметры:
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - указатель на новый экземпляр App
//   - ошибку, если произошла проблема при инициализации
//
// Функция настраивает все необходимые компоненты приложения,
// включая хранилище URL, генератор коротких идентификаторов,
// сервисный слой и обработчики запросов.
func NewApp(cfg *config.Config) (*App, error) {
	urlStorage, err := storage.NewStorage(cfg.DatabaseDSN, cfg.FileStoragePath)
	if err != nil {
		return nil, err
	}

	urlGenerator := generator.NewGenerator(8)

	urlService := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		urlGenerator,
		cfg.BaseURL,
	)

	handler := handler.NewURLHandler(
		urlService,
		urlService,
		urlService,
		urlService,
		urlService,
		urlService,
		cfg.BaseURL,
	)

	return &App{
		Handler: handler,
		Service: urlService,
	}, nil
}
