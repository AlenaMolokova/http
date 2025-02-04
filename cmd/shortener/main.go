package main

import (
	"log"
	"net/http"

	"github.com/AlenaMolokova/http/internal/app"

	"github.com/gorilla/mux"
)

func main() {
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
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("Starting server on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
