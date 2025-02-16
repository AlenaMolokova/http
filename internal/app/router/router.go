package router

import (
    "net/http"
    "github.com/gorilla/mux"
)

type URLHandler interface {
    HandleShortenURL(w http.ResponseWriter, r *http.Request)
    HandleRedirect(w http.ResponseWriter, r *http.Request)
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

    router.HandleFunc("/", r.handler.HandleShortenURL).Methods(http.MethodPost)
    router.HandleFunc("/{id}", r.handler.HandleRedirect).Methods(http.MethodGet)

    router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "Not Found", http.StatusBadRequest)
    })
    router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "Method not allowed", http.StatusBadRequest)
    })

    return router
}