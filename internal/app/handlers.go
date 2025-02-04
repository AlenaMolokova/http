package app

import (
    "io"
    "math/rand"
    "net/http"
    "strings"
    "time"
    "github.com/gorilla/mux"
	"github.com/AlenaMolokova/http/internal/app/config"
)

var urlStorage = make(map[string]string)
const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var cfg *config.Config

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func InitHandlers(config *config.Config) {
    cfg = config
}

func generateShortID() string {
    id := make([]byte, 8)
    for i := range id {
        id[i] = letters[seededRand.Intn(len(letters))]
    }
    return string(id)
}

func HandleShortenURL(w http.ResponseWriter, r *http.Request) {
    if !strings.Contains(r.Header.Get("Content-Type"), "text/plain") {
        http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
        return
    }

    body, err := io.ReadAll(r.Body)
    defer r.Body.Close()

    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }

    url := strings.TrimSpace(string(body))
    if url == "" {
        http.Error(w, "Empty URL", http.StatusBadRequest)
        return
    }

    shortID := generateShortID()
    urlStorage[shortID] = url

	baseURL := "http://localhost:8080" 
    if cfg != nil && cfg.BaseURL != "" {
        baseURL = cfg.BaseURL
    }

	shortURL := cfg.BaseURL + "/" + shortID
    w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(shortURL))
}

func HandleRedirect(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id := vars["id"]

    originalURL, found := urlStorage[id]
    if !found {
        http.Error(w, "URL not found", http.StatusBadRequest)
        return
    }

    w.Header().Set("Location", originalURL)
    w.WriteHeader(http.StatusTemporaryRedirect)
}