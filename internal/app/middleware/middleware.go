// Package middleware содержит обертки для middleware функций.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// contextKey представляет пользовательский тип для ключей контекста.
// Используется для безопасного хранения значений в контексте HTTP-запросов,
// предотвращая коллизии ключей между различными пакетами.
type contextKey string

// userIDKey является ключом для хранения идентификатора пользователя в контексте.
// Используется middleware WithUserID для добавления и извлечения userID.
const userIDKey contextKey = "userID"

// WithLogging создает middleware для логирования HTTP-запросов.
// Оборачивает следующий обработчик, добавляя логирование входящих запросов и ответов.
//
// Параметры:
//   - next: следующий HTTP-обработчик в цепочке
//
// Возвращает:
//   - новый HTTP-обработчик с функциональностью логирования
func WithLogging(next http.Handler) http.Handler {
	return LoggingMiddleware(next)
}

// WithGzipCompression создает middleware для сжатия HTTP-ответов с использованием gzip.
// Оборачивает следующий обработчик, применяя сжатие к ответам, если клиент поддерживает gzip.
//
// Параметры:
//   - next: следующий HTTP-обработчик в цепочке
//
// Возвращает:
//   - новый HTTP-обработчик с функциональностью сжатия
func WithGzipCompression(next http.Handler) http.Handler {
	return GzipMiddleware(next)
}

// WithUserID создает middleware для добавления идентификатора пользователя в контекст запроса.
// Проверяет наличие заголовка X-User-ID в запросе. Если заголовок отсутствует,
// генерирует случайный идентификатор пользователя. Добавляет userID в контекст
// и устанавливает заголовок X-User-ID в ответе.
//
// Параметры:
//   - next: следующий HTTP-обработчик в цепочке
//
// Возвращает:
//   - новый HTTP-обработчик с функциональностью добавления userID
func WithUserID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			// Генерируем случайный User ID если его нет
			userID = generateUserID()
		}

		// Добавляем User ID в контекст
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		r = r.WithContext(ctx)

		// Добавляем User ID в заголовок ответа для отладки
		w.Header().Set("X-User-ID", userID)

		next.ServeHTTP(w, r)
	})
}

// generateUserID генерирует случайный идентификатор пользователя.
// Создает 16-байтный случайный массив и преобразует его в шестнадцатеричную строку.
//
// Возвращает:
//   - строка, представляющая уникальный идентификатор пользователя
func generateUserID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetUserIDFromContext извлекает идентификатор пользователя из контекста HTTP-запроса.
// Если идентификатор отсутствует в контексте, возвращает значение "anonymous".
//
// Параметры:
//   - ctx: контекст HTTP-запроса
//
// Возвращает:
//   - идентификатор пользователя или "anonymous", если userID не найден
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return "anonymous"
}
