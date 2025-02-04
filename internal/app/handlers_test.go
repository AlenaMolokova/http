package app

import (
    "github.com/gorilla/mux"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestHandleShortenURL_Success(t *testing.T) {
    cfg = &config.Config{BaseURL: "http://localhost:8080"}
	
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
    req.Header.Set("Content-Type", "text/plain")
    w := httptest.NewRecorder()

    HandleShortenURL(w, req)

    if w.Code != http.StatusCreated {
        t.Errorf("Expected 201, got %d", w.Code)
    }

    responseBody := w.Body.String()
    if !strings.Contains(responseBody, "http://localhost:8080/") {
        t.Errorf("Expected shortened URL, got %s", responseBody)
    }
}

func TestHandleShortenURL_InvalidContentType(t *testing.T) {
    req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    HandleShortenURL(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected 400, got %d", w.Code)
    }
}

func TestHandleShortenURL_EmptyBody(t *testing.T) {
    req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
    req.Header.Set("Content-Type", "text/plain")
    w := httptest.NewRecorder()

    HandleShortenURL(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected 400, got %d", w.Code)
    }
}

func TestHandleRedirect_Success(t *testing.T) {
    router := mux.NewRouter()
    router.HandleFunc("/{id}", HandleRedirect)

    shortID := generateShortID()
    originalURL := "https://example.com"
    urlStorage[shortID] = originalURL

    req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
    w := httptest.NewRecorder()

    router.ServeHTTP(w, req)

    if w.Code != http.StatusTemporaryRedirect {
        t.Errorf("Expected 307, got %d", w.Code)
    }

    if location := w.Header().Get("Location"); location != originalURL {
        t.Errorf("Expected redirect to %s, got %s", originalURL, location)
    }
}

func TestHandleRedirect_NotFound(t *testing.T) {
    router := mux.NewRouter()
    router.HandleFunc("/{id}", HandleRedirect)

    req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
    w := httptest.NewRecorder()

    router.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected 400, got %d", w.Code)
    }
}