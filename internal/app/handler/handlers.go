package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"fmt"
	
	"github.com/AlenaMolokova/http/internal/app/auth"
	"github.com/AlenaMolokova/http/internal/app/models"
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
	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}
	
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "text/plain") {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logrus.WithError(err).Error("Failed to read request body")
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	if _, err := url.Parse(originalURL); err != nil {
		logrus.WithError(err).Error("Invalid URL format")
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.ShortenURL(originalURL, userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to shorten URL")
		http.Error(w, "Failed to shorten URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if existingShortID, err := h.service.FindByOriginalURL(originalURL); err == nil && existingShortID != "" {
		shortURL = fmt.Sprintf("%s/%s", getBaseURL(r), existingShortID) 
	}
	w.WriteHeader(http.StatusCreated) 
	w.Write([]byte(shortURL))
}

func (h *Handler) HandleShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	var req models.ShortenRequest
	if r.Body == nil {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logrus.WithError(err).Error("Invalid JSON format")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON format"})
		return
	}

	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "URL cannot be empty"})
		return
	}

	if _, err := url.Parse(req.URL); err != nil {
		logrus.WithError(err).Error("Invalid URL format")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid URL format"})
		return
	}

	shortURL, err := h.service.ShortenURL(req.URL, userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to shorten URL")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to shorten URL"})
		return
	}

	if existingShortID, err := h.service.FindByOriginalURL(req.URL); err == nil && existingShortID != "" {
		shortURL = fmt.Sprintf("%s/%s", getBaseURL(r), existingShortID) 
	}
	w.WriteHeader(http.StatusCreated) 
	json.NewEncoder(w).Encode(models.ShortenResponse{Result: shortURL})
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

func (h *Handler) HandlePing(w http.ResponseWriter, r *http.Request) {
	err := h.service.Ping()
	if err != nil {
		if err.Error() == "file storage does not support database connection check" ||
			err.Error() == "memory storage does not support database connection check" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Storage does not require database connection"))
			return
		}

		logrus.WithError(err).Error("Database ping failed")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Database connection is OK"))

}

func (h *Handler) HandleBatchShortenURL(w http.ResponseWriter, r *http.Request) {
	var req []models.BatchShortenRequest

	if r.Body == nil {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logrus.WithError(err).Error("Invalid JSON format")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON format"})
		return
	}

	if len(req) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Empty batch"})
		return
	}

	for _, item := range req {
		if item.OriginalURL == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "URL cannot be empty"})
			return
		}

		if _, err := url.Parse(item.OriginalURL); err != nil {
			logrus.WithError(err).Error("Invalid URL format")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid URL format"})
			return
		}
	}

	userID, err := auth.GetUserIDFromCookie(r)
    if err != nil {
        userID = auth.GenerateUserID()
        auth.SetUserIDCookie(w, userID)
    }

	resp, err := h.service.ShortenBatch(req, userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to shorten batch")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to shorten batch"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}


func (h *Handler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := h.service.GetUserURLs(userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to get user URLs")
		http.Error(w, "Failed to get user URLs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	baseURL := getBaseURL(r)
	for i := range urls {
		urls[i].ShortURL = baseURL + "/" + urls[i].ShortURL
	}

	if err := json.NewEncoder(w).Encode(urls); err != nil {
		logrus.WithError(err).Error("Failed to encode user URLs")
		http.Error(w, "Failed to encode user URLs", http.StatusInternalServerError)
		return
	}
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}