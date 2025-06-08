// Package middleware_test содержит тесты для middleware логирования.
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/middleware"
	"github.com/stretchr/testify/assert"
)

// TestLoggingMiddleware проверяет, что middleware вызывает следующий обработчик и логирует запрос.
func TestLoggingMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()

	var wasCalled bool

	handler := middleware.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wasCalled = true
		w.WriteHeader(http.StatusCreated)
	}))

	handler.ServeHTTP(rr, req)

	assert.True(t, wasCalled, "обработчик не был вызван")
	assert.Equal(t, http.StatusCreated, rr.Code)
}
