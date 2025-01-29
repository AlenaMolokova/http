package main

import (
    "net/http"
    "log"
	"io"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/", handleURL)

    log.Println("Starting server on :8080")
    err := http.ListenAndServe(":8080", mux)
    if err != nil {
        log.Fatal(err)
    }
}

func handleURL(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Only POST method is allowed", http.StatusBadRequest)
        return
    }

    
    if r.URL.Path != "/" {
        http.Error(w, "Invalid path", http.StatusBadRequest)
        return
    }
    
    
