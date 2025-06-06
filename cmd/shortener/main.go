package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlenaMolokova/http/internal/app"
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/sirupsen/logrus"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка запуска: %v\n", err)
		os.Exit(1)
	}
}

// printBuildInfo выводит информацию о сборке приложения.
func printBuildInfo() {
	fmt.Printf("Версия: %s\nДата сборки: %s\nКоммит: %s\n",
		getValue(buildVersion),
		getValue(buildDate),
		getValue(buildCommit))
}

// getValue возвращает значение или "N/A" если значение пустое.
func getValue(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}

// run запускает основную логику приложения.
func run() error {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	cfg := config.NewConfig()

	appInstance, err := app.NewApp(cfg)
	if err != nil {
		return fmt.Errorf("ошибка инициализации: %w", err)
	}

	server := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      appInstance.Handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Канал для graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Канал для ошибок сервера
	serverErr := make(chan error, 1)

	// Запускаем сервер в отдельной горутине
	go func() {
		logrus.Infof("Сервер запущен на %s", cfg.ServerAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("ошибка запуска сервера: %w", err)
		}
	}()

	// Небольшая задержка для старта сервера
	time.Sleep(100 * time.Millisecond)

	// Ожидаем сигнал остановки или ошибку сервера
	select {
	case err := <-serverErr:
		return err
	case sig := <-stop:
		logrus.Infof("Получен сигнал остановки: %v, завершаем работу...", sig)
	}

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем HTTP сервер
	if err := server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Ошибка при остановке сервера")
		return err
	}

	logrus.Info("Сервер остановлен")
	return nil
}
