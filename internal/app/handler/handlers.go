package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/AlenaMolokova/http/internal/app/auth"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type ShortenHandler struct {
	shortener service.Shortener
}

type RedirectHandler struct {
	redirector service.URLGetter
	fetcher    service.URLFetcher
	baseURL    string
}

type UserURLsHandler struct {
	fetcher service.URLFetcher
}

type DeleteHandler struct {
	deleter service.URLDeleter
}

type PingHandler struct {
	pinger service.Pinger
}

type URLHandler struct {
	shorten  *ShortenHandler
	redirect *RedirectHandler
	userURLs *UserURLsHandler
	delete   *DeleteHandler
	ping     *PingHandler
}

func NewShortenHandler(shortener service.Shortener) *ShortenHandler {
	return &ShortenHandler{shortener}
}

func NewRedirectHandler(redirector service.URLGetter, fetcher service.URLFetcher, baseURL string) *RedirectHandler {
	return &RedirectHandler{redirector, fetcher, baseURL}
}

func NewUserURLsHandler(fetcher service.URLFetcher) *UserURLsHandler {
	return &UserURLsHandler{fetcher}
}

func NewDeleteHandler(deleter service.URLDeleter) *DeleteHandler {
	return &DeleteHandler{deleter}
}

func NewPingHandler(pinger service.Pinger) *PingHandler {
	return &PingHandler{pinger}
}

func NewURLHandler(shortener service.Shortener, getter service.URLGetter, fetcher service.URLFetcher, deleter service.URLDeleter, pinger service.Pinger, baseURL string) *URLHandler {
	return &URLHandler{
		shorten:  NewShortenHandler(shortener),
		redirect: NewRedirectHandler(getter, fetcher, baseURL),
		userURLs: NewUserURLsHandler(fetcher),
		delete:   NewDeleteHandler(deleter),
		ping:     NewPingHandler(pinger),
	}
}

func (h *ShortenHandler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
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

	ctx := r.Context()
	result, err := h.shortener.ShortenURL(ctx, originalURL, userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to shorten URL")
		http.Error(w, "Failed to shorten URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if result.IsNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusConflict)
	}
	w.Write([]byte(result.ShortURL))
}

func (h *ShortenHandler) HandleShortenURLJSON(w http.ResponseWriter, r *http.Request) {
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

	ctx := r.Context()
	result, err := h.shortener.ShortenURL(ctx, req.URL, userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to shorten URL")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to shorten URL"})
		return
	}

	resp := models.ShortenResponse{Result: result.ShortURL}
	if result.IsNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *ShortenHandler) HandleBatchShortenURL(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	var req []models.BatchShortenRequest
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

	ctx := r.Context()
	resp, err := h.shortener.ShortenBatch(ctx, req, userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to shorten batch")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to shorten batch"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *RedirectHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	logrus.WithFields(logrus.Fields{
		"id":     id,
		"method": r.Method,
		"uri":    r.RequestURI,
	}).Info("Handling redirect request")

	ctx := r.Context()
	originalURL, found := h.redirector.Get(ctx, id)
	if !found {
		logrus.WithField("id", id).Warn("URL not found or deleted")
		http.Error(w, "Gone", http.StatusGone)
		return
	}

	logrus.WithFields(logrus.Fields{
		"id":          id,
		"redirect_to": originalURL,
	}).Info("Redirecting to original URL")

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *UserURLsHandler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		logrus.WithError(err).Warn("No valid cookie found, generating new user ID")
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	ctx := r.Context()
	urls, err := h.fetcher.GetURLsByUserID(ctx, userID)
	if err != nil {
		logrus.WithError(err).Error("Failed to get user URLs")
		http.Error(w, "Failed to get user URLs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(urls) == 0 {
		logrus.WithField("user_id", userID).Info("No URLs found for user")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := json.NewEncoder(w).Encode(urls); err != nil {
		logrus.WithError(err).Error("Failed to encode user URLs")
		http.Error(w, "Failed to encode user URLs", http.StatusInternalServerError)
		return
	}
}

func (h *DeleteHandler) HandleDeleteURLs(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		logrus.WithError(err).Warn("No valid cookie found, unauthorized")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var shortIDs []string
	if err := json.NewDecoder(r.Body).Decode(&shortIDs); err != nil {
		logrus.WithError(err).Error("Invalid JSON format")
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(shortIDs) == 0 {
		http.Error(w, "Empty list of URLs", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.deleter.DeleteURLs(ctx, shortIDs, userID); err != nil {
		logrus.WithError(err).Error("Failed to delete URLs")
		http.Error(w, "Failed to delete URLs", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *PingHandler) HandlePing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := h.pinger.Ping(ctx)
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

func (h *URLHandler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	h.shorten.HandleShortenURL(w, r)
}

func (h *URLHandler) HandleShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	h.shorten.HandleShortenURLJSON(w, r)
}

func (h *URLHandler) HandleBatchShortenURL(w http.ResponseWriter, r *http.Request) {
	h.shorten.HandleBatchShortenURL(w, r)
}

func (h *URLHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	h.redirect.HandleRedirect(w, r)
}

func (h *URLHandler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	h.userURLs.HandleGetUserURLs(w, r)
}

func (h *URLHandler) HandleDeleteURLs(w http.ResponseWriter, r *http.Request) {
	h.delete.HandleDeleteURLs(w, r)
}

func (h *URLHandler) HandlePing(w http.ResponseWriter, r *http.Request) {
	h.ping.HandlePing(w, r)
}
