package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v9"
)

type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL       string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"urls.json"`
}

func NewConfig() *Config {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		log.Fatalf("Ошибка парсинга конфигурации: %v", err)
	}

	serverAddress := flag.String("a", cfg.ServerAddress, "HTTP server address")
	baseURL := flag.String("b", cfg.BaseURL, "Base URL for shortened URLs")
	fileStoragePath := flag.String("f", cfg.FileStoragePath, "Path for URL storage file")
	
	flag.Parse()

	cfg.ServerAddress = *serverAddress
	cfg.BaseURL = *baseURL
	cfg.FileStoragePath = *fileStoragePath

	return cfg
}
