package config

import (
	"flag"
	"github.com/caarlos0/env/v9"
	"log"
)

type Config struct {
	ServerAddress string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL       string `env:"BASE_URL" envDefault:"http://localhost:8080"`
}

func InitConfig() *Config {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		log.Fatalf("Ошибка парсинга конфигурации: %v", err)
	}

	serverAddress := flag.String("a", cfg.ServerAddress, "HTTP server address")
	baseURL := flag.String("b", cfg.BaseURL, "Base URL for shortened URLs")
	flag.Parse()

	cfg.ServerAddress = *serverAddress
	cfg.BaseURL = *baseURL

	return cfg
}
