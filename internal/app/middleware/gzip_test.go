// Package middleware_test содержит тесты для middleware gzip-сжатия.
package middleware_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/middleware"
	"github.com/stretchr/testify/assert"
)

// TestGzipMiddleware_Decompression проверяет, что middleware корректно распаковывает gzip-тело.
func TestGzipMiddleware_Decompression(t *testing.T) {
	original := "hello, world"
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte(original))
	_ = gz.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/x-gzip")

	rr := httptest.NewRecorder()

	var decoded string
	handler := middleware.GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		decoded = string(body)
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, original, decoded)
}

// TestGzipMiddleware_Compression проверяет, что ответ сервера сжимается, если клиент поддерживает gzip.
func TestGzipMiddleware_Compression(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()

	handler := middleware.GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("response"))
	}))

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

	// Проверим, что тело действительно сжато
	gzReader, err := gzip.NewReader(rr.Body)
	assert.NoError(t, err)

	body, err := io.ReadAll(gzReader)
	assert.NoError(t, err)
	assert.Equal(t, "response", string(body))
}
