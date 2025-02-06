package handler

import "net/http"

type URLHandler interface {
	HandleShortenURL(w http.ResponseWriter, r *http.Request)
	HandleRedirect(w http.ResponseWriter, r *http.Request)
}
