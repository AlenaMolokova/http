// Package config предоставляет функции для конфигурации приложения.
package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/caarlos0/env/v9"
)

// Config представляет конфигурацию приложения.
// Содержит настройки сервера, базовый URL для сокращенных ссылок,
// путь к файлу хранения, строку подключения к базе данных и настройки HTTPS.
type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:"localhost:8080" json:"server_address"`  // Адрес HTTP-сервера
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080" json:"base_url"`       // Базовый URL для сокращенных ссылок
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"urls.json" json:"file_storage_path"` // Путь к файлу хранения URL
	DatabaseDSN     string `env:"DATABASE_DSN" envDefault:"" json:"database_dsn"`                    // Строка подключения к базе данных
	EnableHTTPS     bool   `env:"ENABLE_HTTPS" envDefault:"false" json:"enable_https"`               // Флаг включения HTTPS
	CertFile        string `env:"CERT_FILE" envDefault:"server.crt" json:"cert_file,omitempty"`      // Путь к файлу сертификата
	KeyFile         string `env:"KEY_FILE" envDefault:"server.key" json:"key_file,omitempty"`        // Путь к файлу приватного ключа
}

// loadFromJSON загружает конфигурацию из JSON-файла.
// Если файл не существует или пуст, возвращает пустую конфигурацию без ошибки.
//
// Параметры:
//   - filename: путь к JSON-файлу конфигурации
//
// Возвращает:
//   - указатель на структуру Config с загруженными данными
//   - ошибку, если файл существует, но не может быть прочитан или распарсен
func loadFromJSON(filename string) (*Config, error) {
	if filename == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil // Файл не существует - не ошибка
		}
		return nil, err
	}

	if len(data) == 0 {
		return &Config{}, nil // Пустой файл - не ошибка
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// NewConfig создает и возвращает новый экземпляр конфигурации.
// Функция загружает настройки в следующем порядке приоритета:
// 1. Флаги командной строки (высший приоритет)
// 2. Переменные окружения
// 3. JSON-файл конфигурации
// 4. Значения по умолчанию (низший приоритет)
//
// В случае ошибки при разборе конфигурации, функция завершает работу программы.
//
// Возвращает: указатель на заполненную структуру Config.
func NewConfig() *Config {
	// Создаем конфигурацию со значениями по умолчанию
	defaultCfg := &Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "urls.json",
		DatabaseDSN:     "",
		EnableHTTPS:     false,
		CertFile:        "server.crt",
		KeyFile:         "server.key",
	}

	// Определяем все флаги сразу, включая путь к конфигурационному файлу
	var configPath string
	flag.StringVar(&configPath, "c", "", "Path to JSON configuration file")
	flag.StringVar(&configPath, "config", "", "Path to JSON configuration file")

	serverAddress := flag.String("a", defaultCfg.ServerAddress, "HTTP server address")
	baseURL := flag.String("b", defaultCfg.BaseURL, "Base URL for shortened URLs")
	fileStoragePath := flag.String("f", defaultCfg.FileStoragePath, "Path for URL storage file")
	databaseDSN := flag.String("d", defaultCfg.DatabaseDSN, "Database connection string")
	enableHTTPS := flag.Bool("s", defaultCfg.EnableHTTPS, "Enable HTTPS")
	certFile := flag.String("cert", defaultCfg.CertFile, "Path to TLS certificate file")
	keyFile := flag.String("key", defaultCfg.KeyFile, "Path to TLS private key file")

	flag.Parse()

	// Проверяем переменную окружения CONFIG если путь не указан через флаг
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	// Загружаем из JSON-файла если указан
	jsonCfg, err := loadFromJSON(configPath)
	if err != nil {
		log.Fatalf("Ошибка загрузки JSON конфигурации: %v", err)
	}

	// Начинаем с значений по умолчанию
	cfg := defaultCfg

	// Применяем значения из JSON-файла
	if jsonCfg.ServerAddress != "" {
		cfg.ServerAddress = jsonCfg.ServerAddress
	}
	if jsonCfg.BaseURL != "" {
		cfg.BaseURL = jsonCfg.BaseURL
	}
	if jsonCfg.FileStoragePath != "" {
		cfg.FileStoragePath = jsonCfg.FileStoragePath
	}
	if jsonCfg.DatabaseDSN != "" {
		cfg.DatabaseDSN = jsonCfg.DatabaseDSN
	}
	if jsonCfg.EnableHTTPS {
		cfg.EnableHTTPS = jsonCfg.EnableHTTPS
	}
	if jsonCfg.CertFile != "" {
		cfg.CertFile = jsonCfg.CertFile
	}
	if jsonCfg.KeyFile != "" {
		cfg.KeyFile = jsonCfg.KeyFile
	}

	// Применяем переменные окружения
	envCfg := &Config{}
	if err := env.Parse(envCfg); err != nil {
		log.Fatalf("Ошибка парсинга переменных окружения: %v", err)
	}

	// Применяем непустые значения из переменных окружения
	if envCfg.ServerAddress != "" && envCfg.ServerAddress != "localhost:8080" {
		cfg.ServerAddress = envCfg.ServerAddress
	}
	if envCfg.BaseURL != "" && envCfg.BaseURL != "http://localhost:8080" {
		cfg.BaseURL = envCfg.BaseURL
	}
	if envCfg.FileStoragePath != "" && envCfg.FileStoragePath != "urls.json" {
		cfg.FileStoragePath = envCfg.FileStoragePath
	}
	if envCfg.DatabaseDSN != "" {
		cfg.DatabaseDSN = envCfg.DatabaseDSN
	}
	if envCfg.EnableHTTPS {
		cfg.EnableHTTPS = envCfg.EnableHTTPS
	}
	if envCfg.CertFile != "" && envCfg.CertFile != "server.crt" {
		cfg.CertFile = envCfg.CertFile
	}
	if envCfg.KeyFile != "" && envCfg.KeyFile != "server.key" {
		cfg.KeyFile = envCfg.KeyFile
	}

	// Применяем значения из флагов (высший приоритет)
	// Проверяем, что флаг был явно задан пользователем
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.ServerAddress = *serverAddress
		case "b":
			cfg.BaseURL = *baseURL
		case "f":
			cfg.FileStoragePath = *fileStoragePath
		case "d":
			cfg.DatabaseDSN = *databaseDSN
		case "s":
			cfg.EnableHTTPS = *enableHTTPS
		case "cert":
			cfg.CertFile = *certFile
		case "key":
			cfg.KeyFile = *keyFile
		}
	})

	return cfg
}
