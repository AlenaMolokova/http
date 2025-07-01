// Package middleware содержит обертки для middleware функций.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// WithLogging применяет middleware для логирования.
func WithLogging(next http.Handler) http.Handler {
	return LoggingMiddleware(next)
}

// WithGzipCompression применяет middleware для gzip сжатия.
func WithGzipCompression(next http.Handler) http.Handler {
	return GzipMiddleware(next)
}

// WithUserID применяет middleware для добавления User ID в контекст.
func WithUserID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			// Генерируем случайный User ID если его нет
			userID = generateUserID()
		}

		// Добавляем User ID в контекст
		ctx := context.WithValue(r.Context(), "userID", userID)
		r = r.WithContext(ctx)

		// Добавляем User ID в заголовок ответа для отладки
		w.Header().Set("X-User-ID", userID)

		next.ServeHTTP(w, r)
	})
}

// generateUserID генерирует случайный User ID.
func generateUserID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetUserIDFromContext извлекает User ID из контекста запроса.
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("userID").(string); ok {
		return userID
	}
	return "anonymous"
}
