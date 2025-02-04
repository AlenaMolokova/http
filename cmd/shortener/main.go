package main

import (
	"log"
	"net/http"

	"github.com/AlenaMolokova/http/internal/app"
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/gorilla/mux"
)

func main() {
	cfg := config.InitConfig()

	app.InitHandlers(cfg)

	router := mux.NewRouter()
	router.HandleFunc("/", app.HandleShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/{id}", app.HandleRedirect).Methods(http.MethodGet)

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
