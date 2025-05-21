package main

import (
	"net/http"

	"github.com/AlenaMolokova/http/internal/app"
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/router"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	cfg := config.NewConfig()
	logrus.WithField("config", cfg).Info("Configuration loaded")

	appInstance, err := app.NewApp(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Не удалось инициализировать приложение")
	}
	logrus.Info("Application initialized")

	r := router.NewRouter(appInstance.Handler)

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r.InitRoutes(),
	}
	logrus.WithFields(logrus.Fields{
		"address":  cfg.ServerAddress,
		"base_url": cfg.BaseURL,
	}).Info("Starting server")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.WithError(err).Fatal("Failed to start server")
	}
	logrus.Info("Server is running")
}