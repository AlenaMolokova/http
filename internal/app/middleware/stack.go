// Package middleware содержит middleware стек для приложения.
package middleware

import (
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/config"
)

// MiddlewareStack представляет стек middleware для приложения.
type MiddlewareStack struct {
	config *config.Config
}

// NewMiddlewareStack создает новый стек middleware.
func NewMiddlewareStack(cfg *config.Config) *MiddlewareStack {
	return &MiddlewareStack{
		config: cfg,
	}
}

// Apply применяет все middleware к обработчику.
func (ms *MiddlewareStack) Apply(handler http.Handler) http.Handler {
	// Применяем middleware в обратном порядке (последний применяется первым)
	handler = WithUserID(handler)
	handler = WithGzipCompression(handler)
	handler = WithLogging(handler)

	return handler
}
