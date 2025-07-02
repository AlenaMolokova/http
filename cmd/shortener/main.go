// Package main содержит точку входа для сервиса сокращения URL.
// Запускает HTTP и gRPC серверы с общей бизнес-логикой и хранилищем.
package main

import (
	"log"
	"net/http"

	"github.com/AlenaMolokova/http/internal/app"
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/grpc/server"
	"github.com/sirupsen/logrus"
)

// main запускает приложение, инициализируя конфигурацию, хранилище,
// HTTP и gRPC серверы.
func main() {
	// Инициализация конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Проверка конфигурации
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// Инициализация логгера
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Инициализация приложения
	appInstance, err := app.NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	defer appInstance.Close()

	// Инициализация gRPC-обработчика
	grpcHandler := server.NewShortenerHandler(
		appInstance.Service, // URLShortener
		appInstance.Service, // BatchURLShortener
		appInstance.Service, // URLGetter
		appInstance.Service, // URLFetcher
		appInstance.Service, // URLDeleter
		appInstance.Service, // Pinger
		appInstance.Service, // StatsProvider
		cfg.BaseURL,
	)

	// Запуск gRPC-сервера
	grpcServer, err := server.NewGRPCServer(
		cfg.GetGRPCAddress(),
		grpcHandler,
		cfg,
		logger,
		cfg.IsTLSEnabled(),
		cfg.GetTLSCertFile(),
		cfg.GetTLSKeyFile(),
	)
	if err != nil {
		log.Fatalf("Failed to initialize gRPC server: %v", err)
	}

	go func() {
		if err := grpcServer.Start(); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Запуск HTTP-сервера
	httpServer := &http.Server{
		Addr:    cfg.GetHTTPAddress(),
		Handler: appInstance.Handler.InitRoutes(),
	}

	if cfg.EnableHTTPS {
		if err := httpServer.ListenAndServeTLS(cfg.GetTLSCertFile(), cfg.GetTLSKeyFile()); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	} else {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}
}
