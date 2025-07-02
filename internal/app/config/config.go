// Package config предоставляет функции и структуры для управления конфигурацией приложения.
// Поддерживает загрузку настроек из флагов командной строки, переменных окружения и JSON-файлов.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/caarlos0/env/v9"
)

// Config представляет конфигурацию приложения.
// Содержит настройки для HTTP и gRPC серверов, базовый URL для сокращенных ссылок,
// путь к файлу хранения, строку подключения к базе данных, настройки TLS и доверенную подсеть.
type Config struct {
	// HTTP сервер настройки
	ServerAddress string `env:"SERVER_ADDRESS" envDefault:"localhost:8080" json:"server_address"` // Адрес HTTP-сервера
	BaseURL       string `env:"BASE_URL" envDefault:"http://localhost:8080" json:"base_url"`      // Базовый URL для сокращенных ссылок
	EnableHTTPS   bool   `env:"ENABLE_HTTPS" envDefault:"false" json:"enable_https"`              // Флаг включения HTTPS

	// gRPC сервер настройки
	GRPCAddr string `env:"GRPC_ADDRESS" envDefault:":3201" json:"grpc_address"` // Адрес gRPC-сервера

	// TLS настройки (общие для HTTP и gRPC)
	EnableTLS   bool   `env:"ENABLE_TLS" envDefault:"false" json:"enable_tls"`                      // Флаг включения TLS для gRPC
	TLSCertFile string `env:"TLS_CERT_FILE" envDefault:"server.crt" json:"tls_cert_file,omitempty"` // Путь к файлу TLS сертификата
	TLSKeyFile  string `env:"TLS_KEY_FILE" envDefault:"server.key" json:"tls_key_file,omitempty"`   // Путь к файлу TLS приватного ключа

	// Устаревшие поля для обратной совместимости
	CertFile string `env:"CERT_FILE" envDefault:"server.crt" json:"cert_file,omitempty"` // Путь к файлу сертификата (устарело, используйте TLSCertFile)
	KeyFile  string `env:"KEY_FILE" envDefault:"server.key" json:"key_file,omitempty"`   // Путь к файлу приватного ключа (устарело, используйте TLSKeyFile)

	// Хранилище данных
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"urls.json" json:"file_storage_path"` // Путь к файлу хранения URL
	DatabaseDSN     string `env:"DATABASE_DSN" envDefault:"" json:"database_dsn"`                    // Строка подключения к базе данных

	// Сеть
	TrustedSubnet string `env:"TRUSTED_SUBNET" envDefault:"" json:"trusted_subnet"` // Доверенная подсеть в формате CIDR
}

// loadFromJSON загружает конфигурацию из JSON-файла.
// Если файл не существует или пуст, возвращает конфигурацию по умолчанию без ошибки.
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

// LoadConfig загружает конфигурацию из различных источников.
// Это основная функция для получения конфигурации приложения.
//
// Возвращает:
//   - указатель на заполненную структуру Config
//   - ошибку, если произошла ошибка при загрузке конфигурации
func LoadConfig() (*Config, error) {
	cfg := NewConfig()
	return cfg, nil
}

// NewConfig создает новый экземпляр конфигурации.
// Загружает настройки в следующем порядке приоритета:
// 1. Флаги командной строки (высший приоритет)
// 2. Переменные окружения
// 3. JSON-файл конфигурации
// 4. Значения по умолчанию (низший приоритет)
//
// Завершает выполнение программы в случае ошибки парсинга конфигурации.
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
		GRPCAddr:        ":3201",
		EnableTLS:       false,
		TLSCertFile:     "server.crt",
		TLSKeyFile:      "server.key",
		CertFile:        "server.crt",
		KeyFile:         "server.key",
		TrustedSubnet:   "",
	}

	// Определяем все флаги
	var configPath string
	flag.StringVar(&configPath, "c", "", "Path to JSON configuration file")
	flag.StringVar(&configPath, "config", "", "Path to JSON configuration file")

	// HTTP сервер флаги
	serverAddress := flag.String("a", defaultCfg.ServerAddress, "HTTP server address")
	baseURL := flag.String("b", defaultCfg.BaseURL, "Base URL for shortened URLs")
	enableHTTPS := flag.Bool("s", defaultCfg.EnableHTTPS, "Enable HTTPS")

	// gRPC сервер флаги
	grpcAddr := flag.String("grpc", defaultCfg.GRPCAddr, "gRPC server address")
	enableTLS := flag.Bool("tls", defaultCfg.EnableTLS, "Enable TLS for gRPC")

	// Хранилище флаги
	fileStoragePath := flag.String("f", defaultCfg.FileStoragePath, "Path for URL storage file")
	databaseDSN := flag.String("d", defaultCfg.DatabaseDSN, "Database connection string")

	// TLS сертификаты флаги
	tlsCertFile := flag.String("tls-cert", defaultCfg.TLSCertFile, "Path to TLS certificate file")
	tlsKeyFile := flag.String("tls-key", defaultCfg.TLSKeyFile, "Path to TLS private key file")

	// Устаревшие флаги
	certFile := flag.String("cert", defaultCfg.CertFile, "Path to TLS certificate file (deprecated, use -tls-cert)")
	keyFile := flag.String("key", defaultCfg.KeyFile, "Path to TLS private key file (deprecated, use -tls-key)")

	// Сеть флаг
	trustedSubnet := flag.String("t", defaultCfg.TrustedSubnet, "Trusted subnet in CIDR format")

	flag.Parse()

	// Проверяем переменную окружения CONFIG
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	// Загружаем из JSON-файла
	jsonCfg, err := loadFromJSON(configPath)
	if err != nil {
		log.Fatalf("Ошибка загрузки JSON конфигурации: %v", err)
	}

	// Начинаем с значений по умолчанию
	cfg := defaultCfg

	// Применяем значения из JSON-файла
	applyJSONConfig(cfg, jsonCfg)

	// Применяем переменные окружения
	envCfg := &Config{}
	if err := env.Parse(envCfg); err != nil {
		log.Fatalf("Ошибка парсинга переменных окружения: %v", err)
	}
	applyEnvConfig(cfg, envCfg)

	// Применяем значения из флагов
	applyFlagConfig(cfg, map[string]interface{}{
		"a":        serverAddress,
		"b":        baseURL,
		"f":        fileStoragePath,
		"d":        databaseDSN,
		"s":        enableHTTPS,
		"grpc":     grpcAddr,
		"tls":      enableTLS,
		"tls-cert": tlsCertFile,
		"tls-key":  tlsKeyFile,
		"cert":     certFile,
		"key":      keyFile,
		"t":        trustedSubnet,
	})

	// Синхронизируем устаревшие поля
	syncLegacyFields(cfg)

	return cfg
}

// applyJSONConfig применяет значения из JSON конфигурации к текущей конфигурации.
func applyJSONConfig(cfg, jsonCfg *Config) {
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
	if jsonCfg.GRPCAddr != "" {
		cfg.GRPCAddr = jsonCfg.GRPCAddr
	}
	if jsonCfg.EnableTLS {
		cfg.EnableTLS = jsonCfg.EnableTLS
	}
	if jsonCfg.TLSCertFile != "" {
		cfg.TLSCertFile = jsonCfg.TLSCertFile
	}
	if jsonCfg.TLSKeyFile != "" {
		cfg.TLSKeyFile = jsonCfg.TLSKeyFile
	}
	if jsonCfg.CertFile != "" {
		cfg.CertFile = jsonCfg.CertFile
	}
	if jsonCfg.KeyFile != "" {
		cfg.KeyFile = jsonCfg.KeyFile
	}
	if jsonCfg.TrustedSubnet != "" {
		cfg.TrustedSubnet = jsonCfg.TrustedSubnet
	}
}

// applyEnvConfig применяет значения из переменных окружения к текущей конфигурации.
func applyEnvConfig(cfg, envCfg *Config) {
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
	if envCfg.GRPCAddr != "" && envCfg.GRPCAddr != ":3201" {
		cfg.GRPCAddr = envCfg.GRPCAddr
	}
	if envCfg.EnableTLS {
		cfg.EnableTLS = envCfg.EnableTLS
	}
	if envCfg.TLSCertFile != "" && envCfg.TLSCertFile != "server.crt" {
		cfg.TLSCertFile = envCfg.TLSCertFile
	}
	if envCfg.TLSKeyFile != "" && envCfg.TLSKeyFile != "server.key" {
		cfg.TLSKeyFile = envCfg.TLSKeyFile
	}
	if envCfg.CertFile != "" && envCfg.CertFile != "server.crt" {
		cfg.CertFile = envCfg.CertFile
	}
	if envCfg.KeyFile != "" && envCfg.KeyFile != "server.key" {
		cfg.KeyFile = envCfg.KeyFile
	}
	if envCfg.TrustedSubnet != "" {
		cfg.TrustedSubnet = envCfg.TrustedSubnet
	}
}

// applyFlagConfig применяет значения из флагов командной строки к текущей конфигурации.
func applyFlagConfig(cfg *Config, flags map[string]interface{}) {
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.ServerAddress = *flags["a"].(*string)
		case "b":
			cfg.BaseURL = *flags["b"].(*string)
		case "f":
			cfg.FileStoragePath = *flags["f"].(*string)
		case "d":
			cfg.DatabaseDSN = *flags["d"].(*string)
		case "s":
			cfg.EnableHTTPS = *flags["s"].(*bool)
		case "grpc":
			cfg.GRPCAddr = *flags["grpc"].(*string)
		case "tls":
			cfg.EnableTLS = *flags["tls"].(*bool)
		case "tls-cert":
			cfg.TLSCertFile = *flags["tls-cert"].(*string)
		case "tls-key":
			cfg.TLSKeyFile = *flags["tls-key"].(*string)
		case "cert":
			cfg.CertFile = *flags["cert"].(*string)
		case "key":
			cfg.KeyFile = *flags["key"].(*string)
		case "t":
			cfg.TrustedSubnet = *flags["t"].(*string)
		}
	})
}

// syncLegacyFields синхронизирует устаревшие поля с новыми для обратной совместимости.
func syncLegacyFields(cfg *Config) {
	if cfg.TLSCertFile == "server.crt" && cfg.CertFile != "server.crt" {
		cfg.TLSCertFile = cfg.CertFile
	}
	if cfg.TLSKeyFile == "server.key" && cfg.KeyFile != "server.key" {
		cfg.TLSKeyFile = cfg.KeyFile
	}
	cfg.CertFile = cfg.TLSCertFile
	cfg.KeyFile = cfg.TLSKeyFile
}

// GetHTTPAddress возвращает адрес HTTP-сервера.
// Если адрес не начинается с ':', возвращает его как есть.
//
// Возвращает: полный адрес HTTP-сервера.
func (c *Config) GetHTTPAddress() string {
	if c.ServerAddress == "" {
		return "localhost:8080"
	}
	return c.ServerAddress
}

// GetGRPCAddress возвращает адрес gRPC-сервера.
// Если адрес не начинается с ':', добавляет 'localhost:'.
//
// Возвращает: полный адрес gRPC-сервера.
func (c *Config) GetGRPCAddress() string {
	if c.GRPCAddr == "" {
		return ":3201"
	}
	if len(c.GRPCAddr) > 0 && !strings.HasPrefix(c.GRPCAddr, ":") && !strings.HasPrefix(c.GRPCAddr, "tcp:") {
		return "localhost:" + c.GRPCAddr
	}
	return c.GRPCAddr
}

// GetTLSCertFile возвращает путь к файлу TLS-сертификата.
// Проверяет как новое, так и устаревшее поле для обратной совместимости.
//
// Возвращает: путь к файлу сертификата.
func (c *Config) GetTLSCertFile() string {
	if c.TLSCertFile != "" {
		return c.TLSCertFile
	}
	return c.CertFile
}

// GetTLSKeyFile возвращает путь к файлу TLS-приватного ключа.
// Проверяет как новое, так и устаревшее поле для обратной совместимости.
//
// Возвращает: путь к файлу приватного ключа.
func (c *Config) GetTLSKeyFile() string {
	if c.TLSKeyFile != "" {
		return c.TLSKeyFile
	}
	return c.KeyFile
}

// IsTLSEnabled проверяет, включен ли TLS для gRPC или HTTPS.
// Учитывает как новые, так и устаревшие настройки.
//
// Возвращает: true, если TLS или HTTPS включены.
func (c *Config) IsTLSEnabled() bool {
	return c.EnableTLS || c.EnableHTTPS
}

// HasValidTLSConfig проверяет, есть ли валидная конфигурация TLS.
// Проверяет наличие файлов сертификата и ключа.
//
// Возвращает: true, если конфигурация TLS валидна.
func (c *Config) HasValidTLSConfig() bool {
	certFile := c.GetTLSCertFile()
	keyFile := c.GetTLSKeyFile()
	return certFile != "" && keyFile != ""
}

// Validate проверяет валидность конфигурации.
// Проверяет основные настройки и наличие TLS-файлов, если TLS включен.
//
// Возвращает: ошибку, если конфигурация невалидна.
func (c *Config) Validate() error {
	if c.ServerAddress == "" {
		return log.New(os.Stderr, "", 0).Output(1, "HTTP server address cannot be empty")
	}
	if c.GRPCAddr == "" {
		return log.New(os.Stderr, "", 0).Output(1, "gRPC server address cannot be empty")
	}
	if c.BaseURL == "" {
		return log.New(os.Stderr, "", 0).Output(1, "Base URL cannot be empty")
	}
	if c.IsTLSEnabled() && !c.HasValidTLSConfig() {
		return log.New(os.Stderr, "", 0).Output(1, "TLS is enabled but certificate or key file is missing")
	}
	return nil
}

// String возвращает строковое представление конфигурации.
// Скрывает чувствительные данные, такие как DatabaseDSN.
//
// Возвращает: строковое представление конфигурации.
func (c *Config) String() string {
	maskedDSN := ""
	if c.DatabaseDSN != "" {
		maskedDSN = "***masked***"
	}
	return fmt.Sprintf(`Config{
    HTTP: %s (HTTPS: %t)
    gRPC: %s (TLS: %t)
    BaseURL: %s
    FileStorage: %s
    DatabaseDSN: %s
    TLS Cert: %s
    TLS Key: %s
    TrustedSubnet: %s
}`,
		c.GetHTTPAddress(),
		c.EnableHTTPS,
		c.GetGRPCAddress(),
		c.EnableTLS,
		c.BaseURL,
		c.FileStoragePath,
		maskedDSN,
		c.GetTLSCertFile(),
		c.GetTLSKeyFile(),
		c.TrustedSubnet,
	)
}
