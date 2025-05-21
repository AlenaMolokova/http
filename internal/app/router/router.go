package router

import (
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/middleware"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Router представляет маршрутизатор запросов для сервиса сокращения URL.
type Router struct {
	handler *handler.URLHandler
}

// NewRouter создает новый экземпляр Router с указанным обработчиком.
// Параметр handler - обработчик URL, который будет использоваться для обработки запросов.
func NewRouter(handler *handler.URLHandler) *Router {
	return &Router{
		handler: handler,
	}
}

// InitRoutes инициализирует маршруты для приложения и возвращает настроенный экземпляр маршрутизатора.
// Настраивает все доступные эндпоинты, включая обработку сокращения URL, пакетного сокращения,
// получения URL пользователя, удаления URL, перенаправления и проверки доступности.
func (r *Router) InitRoutes() *mux.Router {
	router := mux.NewRouter()

	router.Use(middleware.GzipMiddleware)
	router.Use(middleware.LoggingMiddleware)

	router.HandleFunc("/", r.handler.HandleShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", r.handler.HandleShortenURLJSON).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten/batch", r.handler.HandleBatchShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/user/urls", r.handler.HandleGetUserURLs).Methods(http.MethodGet)
	router.HandleFunc("/api/user/urls", r.handler.HandleDeleteURLs).Methods(http.MethodDelete)
	router.HandleFunc("/ping", r.handler.HandlePing).Methods(http.MethodGet)
	router.HandleFunc("/{id}", r.handler.HandleRedirect).Methods(http.MethodGet)

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logrus.WithFields(logrus.Fields{
			"uri":    r.RequestURI,
			"method": r.Method,
		}).Info("Route not found")
		http.Error(w, "Not Found", http.StatusBadRequest)
	})

	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logrus.WithFields(logrus.Fields{
			"uri":    r.RequestURI,
			"method": r.Method,
		}).Info("Method not allowed")
		http.Error(w, "Method not allowed", http.StatusBadRequest)
	})

	return router
}
