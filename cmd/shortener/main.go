package main

import (
	"log"
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/gorilla/mux"
)

func main() {
	cfg := config.InitConfig()

	urlStorage := memory.NewMemoryStorage()
	urlGenerator := generator.NewSimpleGenerator(8)
	urlService := service.NewURLService(urlStorage, urlGenerator, cfg.BaseURL)
	urlHandler := handler.NewHandler(urlService)

	router := mux.NewRouter()
	router.HandleFunc("/", urlHandler.HandleShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/{id}", urlHandler.HandleRedirect).Methods(http.MethodGet)

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusBadRequest)
	})
	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method not allowed", http.StatusBadRequest)
	})

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}

	log.Printf("Starting server on %s\n", cfg.ServerAddress)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
