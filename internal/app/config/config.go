// Package config предоставляет функции для конфигурации приложения.
package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v9"
)

// Config представляет конфигурацию приложения.
// Содержит настройки сервера, базовый URL для сокращенных ссылок,
// путь к файлу хранения, строку подключения к базе данных и настройки HTTPS.
type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`  // Адрес HTTP-сервера
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080"` // Базовый URL для сокращенных ссылок
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"urls.json"`    // Путь к файлу хранения URL
	DatabaseDSN     string `env:"DATABASE_DSN" envDefault:""`                  // Строка подключения к базе данных
	EnableHTTPS     bool   `env:"ENABLE_HTTPS" envDefault:"false"`             // Флаг включения HTTPS
	CertFile        string `env:"CERT_FILE" envDefault:"server.crt"`           // Путь к файлу сертификата
	KeyFile         string `env:"KEY_FILE" envDefault:"server.key"`            // Путь к файлу приватного ключа
}

// NewConfig создает и возвращает новый экземпляр конфигурации.
// Функция загружает настройки из переменных окружения и флагов командной строки.
// Приоритет имеют значения, указанные через флаги командной строки.
// В случае ошибки при разборе конфигурации, функция завершает работу программы.
//
// Возвращает: указатель на заполненную структуру Config.
func NewConfig() *Config {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		log.Fatalf("Ошибка парсинга конфигурации: %v", err)
	}

	serverAddress := flag.String("a", cfg.ServerAddress, "HTTP server address")
	baseURL := flag.String("b", cfg.BaseURL, "Base URL for shortened URLs")
	fileStoragePath := flag.String("f", cfg.FileStoragePath, "Path for URL storage file")
	databaseDSN := flag.String("d", cfg.DatabaseDSN, "Database connection string")
	enableHTTPS := flag.Bool("s", cfg.EnableHTTPS, "Enable HTTPS")
	certFile := flag.String("cert", cfg.CertFile, "Path to TLS certificate file")
	keyFile := flag.String("key", cfg.KeyFile, "Path to TLS private key file")

	flag.Parse()

	cfg.ServerAddress = *serverAddress
	cfg.BaseURL = *baseURL
	cfg.FileStoragePath = *fileStoragePath
	cfg.DatabaseDSN = *databaseDSN
	cfg.EnableHTTPS = *enableHTTPS
	cfg.CertFile = *certFile
	cfg.KeyFile = *keyFile

	return cfg
}
