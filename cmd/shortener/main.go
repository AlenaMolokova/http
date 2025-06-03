package main

import (
	"fmt"
	"net/http"
	"os"

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
	}
}

func printBuildInfo() {
	fmt.Printf("Версия: %s\nДата сборки: %s\nКоммит: %s\n",
		getValue(buildVersion),
		getValue(buildDate),
		getValue(buildCommit))
}

func getValue(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}

func run() error {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	cfg := config.NewConfig()

	appInstance, err := app.NewApp(cfg)
	if err != nil {
		return fmt.Errorf("ошибка инициализации: %w", err)
	}

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: appInstance.Handler,
	}

	return server.ListenAndServe()
}
