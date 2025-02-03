package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleShortenURL_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
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

func TestHandleShortenURL_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	HandleShortenURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleShortenURL_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	HandleShortenURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleRedirect_Success(t *testing.T) {
	shortID := "abc123"
	originalURL := "https://example.com"
	urlStorage[shortID] = originalURL

	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w := httptest.NewRecorder()

	HandleRedirect(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected 307, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if location != originalURL {
		t.Errorf("Expected redirect to %s, got %s", originalURL, location)
	}
}

func TestHandleRedirect_InvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/invalidID", nil)
	w := httptest.NewRecorder()

	HandleRedirect(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleRedirect_EmptyID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	HandleRedirect(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleRedirect_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/abc123", nil)
	w := httptest.NewRecorder()

	HandleRedirect(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}
