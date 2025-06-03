// Package handler реализует обработчики HTTP-запросов для приложения.
package handler

import (
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/gorilla/mux"
)

// URLHandler представляет собой основной обработчик URL,
// содержащий все специализированные обработчики и роутер.
type URLHandler struct {
	router          *mux.Router
	ShortenHandler  *ShortenHandler
	RedirectHandler *RedirectHandler
	UserURLsHandler *UserURLsHandler
	DeleteHandler   *DeleteHandler
	PingHandler     *PingHandler
}

// NewURLHandler создает новый основной обработчик URL со всеми подобработчиками.
//
// Параметры:
//   - shortener: сервис для сокращения URL
//   - batch: сервис для пакетного сокращения URL
//   - getter: сервис для получения оригинальных URL
//   - fetcher: сервис для получения URL пользователя
//   - deleter: сервис для удаления URL
//   - pinger: сервис для проверки соединения
//   - baseURL: базовый URL сервиса
//
// Возвращает:
//   - *URLHandler: новый основной обработчик
func NewURLHandler(
	shortener models.URLShortener,
	batch models.BatchURLShortener,
	getter models.URLGetter,
	fetcher models.URLFetcher,
	deleter models.URLDeleter,
	pinger models.Pinger,
	baseURL string,
) *URLHandler {
	// Создаем отдельные обработчики
	shortenHandler := &ShortenHandler{
		shortener: shortener,
		batch:     batch,
		baseURL:   baseURL,
	}
	redirectHandler := &RedirectHandler{
		redirector: getter,
	}
	userURLsHandler := &UserURLsHandler{
		fetcher: fetcher,
	}
	deleteHandler := &DeleteHandler{
		deleter: deleter,
	}
	pingHandler := &PingHandler{
		pinger: pinger,
	}

	// Создаем роутер напрямую
	mainRouter := mux.NewRouter()

	// Настраиваем маршруты для сокращения URL
	shortenRouter := mainRouter.PathPrefix("/").Subrouter()
	shortenRouter.HandleFunc("/", shortenHandler.HandleShortenURL).Methods("POST")
	shortenRouter.HandleFunc("/api/shorten", shortenHandler.HandleShortenURLJSON).Methods("POST")
	shortenRouter.HandleFunc("/api/shorten/batch", shortenHandler.HandleBatchShortenURL).Methods("POST")

	// Настраиваем маршруты для перенаправления
	redirectRouter := mainRouter.PathPrefix("/").Subrouter()
	redirectRouter.HandleFunc("/{id}", redirectHandler.HandleRedirect).Methods("GET")

	// Настраиваем маршруты для пользователей
	userRouter := mainRouter.PathPrefix("/api/user").Subrouter()
	userRouter.HandleFunc("/urls", userURLsHandler.HandleGetUserURLs).Methods("GET")
	userRouter.HandleFunc("/urls", deleteHandler.HandleDeleteURLs).Methods("DELETE")

	// Настраиваем маршрут для проверки соединения
	healthRouter := mainRouter.PathPrefix("/").Subrouter()
	healthRouter.HandleFunc("/ping", pingHandler.HandlePing).Methods("GET")

	return &URLHandler{
		router:          mainRouter,
		ShortenHandler:  shortenHandler,
		RedirectHandler: redirectHandler,
		UserURLsHandler: userURLsHandler,
		DeleteHandler:   deleteHandler,
		PingHandler:     pingHandler,
	}
}

// ServeHTTP реализует интерфейс http.Handler для URLHandler.
func (h *URLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}
