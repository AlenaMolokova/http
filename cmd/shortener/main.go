package main

import (
	"log"
	"net/http"

	"http/internal/app" 
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.Method == http.MethodPost {
			app.HandleShortenURL(w, r)
		} else if r.Method == http.MethodGet {
			app.HandleRedirect(w, r)
		} else {
			http.Error(w, "Invalid request", http.StatusBadRequest)
		}
	})

	log.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
