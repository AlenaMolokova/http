package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	service service.URLService
}

func NewHandler(service service.URLService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Content-Type"), "text/plain") {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.ShortenURL(originalURL)
	if err != nil {
		http.Error(w, "Failed to shorten URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (h *Handler) HandleShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest

	if r.Body == nil {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()


	err :=json.NewDecoder(r.Body).Decode(&req)
	if err !=nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	
	if req.URL =="" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "URL cannot be empty"})
		return
	}

	shortURL, err :=h.service.ShortenURL(req.URL)
	if err !=nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to shorten URL"})
		return
	}

	resp:=ShortenResponse{
		Result: shortURL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	logrus.WithFields(logrus.Fields{
		"id":     id,
		"method": r.Method,
		"uri":    r.RequestURI,
	}).Info("Handling redirect request")

	originalURL, found := h.service.GetOriginalURL(id)
	if !found {
		logrus.WithField("id", id).Warn("URL not found")
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"id":          id,
		"redirect_to": originalURL,
	}).Info("Redirecting to original URL")

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
