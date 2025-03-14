package router

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
    "github.com/AlenaMolokova/http/internal/app/middleware"
)

type URLHandler interface {
	HandleShortenURL(w http.ResponseWriter, r *http.Request)
	HandleShortenURLJSON(w http.ResponseWriter, r *http.Request)
	HandleBatchShortenURL(w http.ResponseWriter, r *http.Request)
	HandleRedirect(w http.ResponseWriter, r *http.Request)
	HandlePing(w http.ResponseWriter, r *http.Request)
}

type Router struct {
	handler URLHandler
}

func NewRouter(handler URLHandler) *Router {
	return &Router{
		handler: handler,
	}
}

func (r *Router) InitRoutes() *mux.Router {
	router := mux.NewRouter()

	router.Use(middleware.GzipMiddleware)
    router.Use(middleware.LoggingMiddleware)

	router.HandleFunc("/", r.handler.HandleShortenURL).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten", r.handler.HandleShortenURLJSON).Methods(http.MethodPost)
	router.HandleFunc("/api/shorten/batch", r.handler.HandleBatchShortenURL).Methods(http.MethodPost)
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
