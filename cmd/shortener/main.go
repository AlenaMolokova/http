package main

import (
	"log"
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/router"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
)

func main() {
	cfg := config.InitConfig()

	urlStorage := memory.NewMemoryStorage()
	urlGenerator := generator.NewSimpleGenerator(8)
	urlService := service.NewURLService(urlStorage, urlGenerator, cfg.BaseURL)
	urlHandler := handler.NewHandler(urlService)
	urlRouter := router.NewRouter(urlHandler)

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: urlRouter.InitRoutes(),
	}

	log.Printf("Starting server on %s\n", cfg.ServerAddress)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
