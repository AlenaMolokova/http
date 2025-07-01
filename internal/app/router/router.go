// Package router предоставляет маршрутизатор для обработки HTTP-запросов в приложении сокращения URL.
// Он настраивает маршруты для операций сокращения, перенаправления, управления пользовательскими URL
// и получения статистики сервиса, используя библиотеку Gorilla Mux.
package router

import (
	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/gorilla/mux"
)

// Router реализует главный роутер приложения с настроенными маршрутами.
// Создается через конструктор NewRouter для обеспечения согласованности.
type Router struct {
	*mux.Router
}

// NewRouter создаёт и инициализирует новый маршрутизатор для обработки HTTP-запросов.
//
// Параметры:
//   - shortener: интерфейс для сокращения URL
//   - batch: интерфейс для пакетного сокращения URL
//   - getter: интерфейс для получения URL
//   - fetcher: интерфейс для получения URL пользователя
//   - deleter: интерфейс для удаления URL
//   - pinger: интерфейс для проверки доступности сервиса
//   - statsProvider: интерфейс для получения статистики
//   - baseURL: базовый URL для формирования сокращённых ссылок
//
// Возвращает:
//   - указатель на Router, готовый к обработке запросов
func NewRouter(
	shortener models.URLShortener,
	batch models.BatchURLShortener,
	getter models.URLGetter,
	fetcher models.URLFetcher,
	deleter models.URLDeleter,
	pinger models.Pinger,
	statsProvider models.StatsProvider,
	baseURL string,
) *Router {
	r := mux.NewRouter()

	// Создаем объединенный обработчик
	urlHandler := handler.NewURLHandler(
		shortener,
		batch,
		getter,
		fetcher,
		deleter,
		pinger,
		statsProvider,
		baseURL,
	)

	// Регистрируем маршруты
	setupRoutes(r, urlHandler)

	return &Router{Router: r}
}

// setupRoutes настраивает все маршруты приложения.
//
// Параметры:
//   - r: маршрутизатор Gorilla Mux для настройки маршрутов
//   - h: обработчик URL для привязки к маршрутам
func setupRoutes(r *mux.Router, h *handler.URLHandler) {
	// Проверка здоровья сервиса
	r.HandleFunc("/ping", h.PingHandler.HandlePing).Methods("GET")

	// Основные маршруты сокращения URL
	r.HandleFunc("/", h.ShortenHandler.HandleShortenURL).Methods("POST")
	r.HandleFunc("/api/shorten", h.ShortenHandler.HandleShortenURLJSON).Methods("POST")
	r.HandleFunc("/api/shorten/batch", h.ShortenHandler.HandleBatchShortenURL).Methods("POST")

	// Перенаправление по короткому URL
	r.HandleFunc("/{id}", h.RedirectHandler.HandleRedirect).Methods("GET")

	// Пользовательские URL
	r.HandleFunc("/api/user/urls", h.UserURLsHandler.HandleGetUserURLs).Methods("GET")
	r.HandleFunc("/api/user/urls", h.DeleteHandler.HandleDeleteURLs).Methods("DELETE")

	// Внутренние API
	r.HandleFunc("/api/internal/stats", h.StatsHandler.HandleStats).Methods("GET")
}

// NewRouterLegacy создаёт устаревший маршрутизатор для обратной совместимости.
//
// Использует отдельные обработчики для каждого типа запросов вместо единого URLHandler.
// Параметры:
//   - shortener: интерфейс для сокращения URL
//   - batch: интерфейс для пакетного сокращения URL
//   - getter: интерфейс для получения URL
//   - fetcher: интерфейс для получения URL пользователя
//   - deleter: интерфейс для удаления URL
//   - pinger: интерфейс для проверки доступности сервиса
//   - baseURL: базовый URL для формирования сокращённых ссылок
//
// Возвращает:
//   - указатель на Router, готовый к обработке запросов
func NewRouterLegacy(
	shortener models.URLShortener,
	batch models.BatchURLShortener,
	getter models.URLGetter,
	fetcher models.URLFetcher,
	deleter models.URLDeleter,
	pinger models.Pinger,
	baseURL string,
) *Router {
	r := mux.NewRouter()

	shortenHandler := handler.NewShortenHandler(shortener, batch, baseURL)
	redirectHandler := handler.NewRedirectHandler(getter)
	userURLsHandler := handler.NewUserURLsHandler(fetcher)
	deleteHandler := handler.NewDeleteHandler(deleter)
	pingHandler := handler.NewPingHandler(pinger)

	r.HandleFunc("/ping", pingHandler.HandlePing).Methods("GET")
	r.HandleFunc("/", shortenHandler.HandleShortenURL).Methods("POST")
	r.HandleFunc("/api/shorten", shortenHandler.HandleShortenURLJSON).Methods("POST")
	r.HandleFunc("/api/shorten/batch", shortenHandler.HandleBatchShortenURL).Methods("POST")
	r.HandleFunc("/{id}", redirectHandler.HandleRedirect).Methods("GET")
	r.HandleFunc("/api/user/urls", userURLsHandler.HandleGetUserURLs).Methods("GET")
	r.HandleFunc("/api/user/urls", deleteHandler.HandleDeleteURLs).Methods("DELETE")

	return &Router{Router: r}
}
