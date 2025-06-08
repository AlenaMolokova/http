// Package handler реализует обработчики HTTP-запросов для сервиса сокращения URL.
package handler

import (
	"github.com/AlenaMolokova/http/internal/app/models"
)

// URLHandler объединяет все обработчики сервиса.
//
// Используется для регистрации всех эндпоинтов в роутере.
type URLHandler struct {
	ShortenHandler  *ShortenHandler  // Обработчик сокращения URL
	RedirectHandler *RedirectHandler // Обработчик перенаправления по короткому URL
	UserURLsHandler *UserURLsHandler // Обработчик получения URL пользователя
	DeleteHandler   *DeleteHandler   // Обработчик удаления URL
	PingHandler     *PingHandler     // Обработчик проверки соединения с хранилищем
}

// NewURLHandler создает объединённый хендлер для всех маршрутов сервиса.
//
// Параметры:
//   - shortener: интерфейс сокращения URL
//   - batch: интерфейс пакетного сокращения URL
//   - getter: интерфейс получения оригинальных URL по коротким
//   - fetcher: интерфейс получения всех URL пользователя
//   - deleter: интерфейс удаления URL
//   - pinger: интерфейс проверки соединения с хранилищем
//   - baseURL: базовый URL приложения
//
// Возвращает:
//   - *URLHandler: агрегированный обработчик
func NewURLHandler(
	shortener models.URLShortener,
	batch models.BatchURLShortener,
	getter models.URLGetter,
	fetcher models.URLFetcher,
	deleter models.URLDeleter,
	pinger models.Pinger,
	baseURL string,
) *URLHandler {
	return &URLHandler{
		ShortenHandler:  NewShortenHandler(shortener, batch, baseURL),
		RedirectHandler: NewRedirectHandler(getter),
		UserURLsHandler: NewUserURLsHandler(fetcher),
		DeleteHandler:   NewDeleteHandler(deleter),
		PingHandler:     NewPingHandler(pinger),
	}
}
