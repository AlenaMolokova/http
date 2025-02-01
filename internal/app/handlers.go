package app

import (
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var urlStorage = make(map[string]string)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateShortID() string {
	rand.Seed(time.Now().UnixNano())
	id := make([]byte, 6)
	for i := range id {
		id[i] = letters[rand.Intn(len(letters))]
	}
	return string(id)
}

func HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	url := strings.TrimSpace(string(body))
	if url == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	shortID := generateShortID()
	urlStorage[shortID] = url

	shortURL := "http://localhost:8080/" + shortID
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func HandleRedirect(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	originalURL, found := urlStorage[id]
	if !found {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}
