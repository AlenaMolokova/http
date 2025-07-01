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

// NewRouter создает новый экземпляр роутера с инициализированными обработчиками.
// Параметры:
// - shortener: сервис сокращения URL
// - batch: сервис пакетной обработки
// - getter: сервис получения URL
// - fetcher: сервис получения пользовательских URL
// - deleter: сервис удаления URL
// - pinger: сервис проверки здоровья
// - statsProvider: сервис получения статистики
// - baseURL: базовый адрес сервиса
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

// setupRoutes настраивает все маршруты приложения
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

// NewRouterLegacy создает роутер

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
