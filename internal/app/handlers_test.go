package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleShortenURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	w := httptest.NewRecorder()

	HandleShortenURL(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Ожидался статус 201, а получили %d", w.Code)
	}

	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "http://localhost:8080/") {
		t.Errorf("Ожидали сокращённый URL, а получили: %s", responseBody)
	}
}

func TestHandleRedirect(t *testing.T) {
	shortID := "abc123"
	originalURL := "https://example.com"
	urlStorage[shortID] = originalURL

	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w := httptest.NewRecorder()

	HandleRedirect(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Ожидался статус 307, а получили %d", w.Code)
	}

	location := w.Header().Get("Location")
	if location != originalURL {
		t.Errorf("Ожидали редирект на %s, а получили %s", originalURL, location)
	}
}
