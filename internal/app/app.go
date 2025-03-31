package app

import (
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage"
)

type App struct {
	Handler *handler.URLHandler 
}

func NewApp(cfg *config.Config) (*App, error) {
	urlStorage, err := storage.NewStorage(cfg.DatabaseDSN, cfg.FileStoragePath)
	if err != nil {
		return nil, err
	}

	urlGenerator := generator.NewGenerator(8)

	urlService := service.NewService(
		urlStorage.Saver,
		urlStorage.BatchSaver,
		urlStorage.Getter,
		urlStorage.Fetcher,
		urlStorage.Deleter,
		urlStorage.Pinger,
		urlGenerator,
		cfg.BaseURL,
	)

	handler := handler.NewURLHandler(urlService, urlService, urlService, urlService, urlService, cfg.BaseURL)

	return &App{
		Handler: handler,
	}, nil
}