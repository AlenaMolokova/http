package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// addAuthCookies добавляет корректные аутентификационные cookies к запросу.
func addAuthCookies(req *http.Request, userID string) {
	secretKey := []byte("your-secret-key-change-this-in-production")
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(userID))
	signature := hex.EncodeToString(h.Sum(nil))

	req.AddCookie(&http.Cookie{Name: "user_id_id", Value: userID})
	req.AddCookie(&http.Cookie{Name: "user_id_sign", Value: signature})
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "1"})
}

// mockPinger реализует интерфейс Pinger для тестов.
type mockPinger struct {
	err error
}

func (m mockPinger) Ping(_ context.Context) error {
	return m.err
}

// mockURLDeleter реализует интерфейс URLDeleter для тестов.
type mockURLDeleter struct {
	err error
}

func (m mockURLDeleter) DeleteURLs(_ context.Context, _ []string, _ string) error {
	return m.err
}

// mockURLGetter реализует интерфейс URLGetter для тестов.
type mockURLGetter struct {
	url    string
	exists bool
}

func (m mockURLGetter) Get(_ context.Context, _ string) (string, bool) {
	return m.url, m.exists
}

// TestHandleDeleteURLsInternalError проверяет поведение обработчика при ошибке удаления URL.
func TestHandleDeleteURLsInternalError(t *testing.T) {
	handler := NewDeleteHandler(mockURLDeleter{err: errors.New("db error")})

	body, _ := json.Marshal([]string{"abc"})
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	addAuthCookies(req, "user42")

	w := httptest.NewRecorder()
	handler.HandleDeleteURLs(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

// TestHandleDeleteURLsEmptyList проверяет поведение обработчика при пустом списке URL.
func TestHandleDeleteURLsEmptyList(t *testing.T) {
	handler := NewDeleteHandler(mockURLDeleter{})

	body, _ := json.Marshal([]string{})
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	addAuthCookies(req, "user42")

	w := httptest.NewRecorder()
	handler.HandleDeleteURLs(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// TestHandlePingMethodNotAllowed проверяет отказ на методы, отличные от GET.
func TestHandlePingMethodNotAllowed(t *testing.T) {
	handler := NewPingHandler(mockPinger{})

	req := httptest.NewRequest(http.MethodPost, "/ping", nil)
	w := httptest.NewRecorder()
	handler.HandlePing(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

// TestHandlePingDatabaseError проверяет ответ 500 при ошибке соединения с базой.
func TestHandlePingDatabaseError(t *testing.T) {
	handler := NewPingHandler(mockPinger{err: errors.New("db down")})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	handler.HandlePing(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

// TestHandlePingFileStorageSkip проверяет, что ответ корректен при неподдерживаемом ping.
func TestHandlePingFileStorageSkip(t *testing.T) {
	handler := NewPingHandler(mockPinger{err: errors.New("file storage does not support database connection check")})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	handler.HandlePing(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Storage does not require database connection") {
		t.Errorf("Expected explanation message, got %s", w.Body.String())
	}
}

// TestHandleRedirectFound проверяет поведение при успешном редиректе.
func TestHandleRedirectFound(t *testing.T) {
	handler := NewRedirectHandler(mockURLGetter{url: "https://example.com", exists: true})

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc123"})
	w := httptest.NewRecorder()

	handler.HandleRedirect(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected 307, got %d", w.Code)
	}
	if location := w.Header().Get("Location"); location != "https://example.com" {
		t.Errorf("Expected Location header https://example.com, got %s", location)
	}
}

// TestHandleRedirectGone проверяет поведение при отсутствии записи.
func TestHandleRedirectGone(t *testing.T) {
	handler := NewRedirectHandler(mockURLGetter{exists: false})

	req := httptest.NewRequest(http.MethodGet, "/gone", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "gone"})
	w := httptest.NewRecorder()

	handler.HandleRedirect(w, req)

	if w.Code != http.StatusGone {
		t.Errorf("Expected 410, got %d", w.Code)
	}
}
