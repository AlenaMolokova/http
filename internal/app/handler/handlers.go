package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/AlenaMolokova/http/internal/app/auth"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/gorilla/mux"
)

type ShortenHandler struct {
	shortener models.URLShortener
	batch     models.BatchURLShortener
	baseURL   string
}

type RedirectHandler struct {
	redirector models.URLGetter
	fetcher    models.URLFetcher
	baseURL    string
}

type UserURLsHandler struct {
	fetcher models.URLFetcher
}

type DeleteHandler struct {
	deleter models.URLDeleter
}

type PingHandler struct {
	pinger models.Pinger
}

type URLHandler struct {
	shorten  *ShortenHandler
	redirect *RedirectHandler
	userURLs *UserURLsHandler
	delete   *DeleteHandler
	ping     *PingHandler
}

func NewShortenHandler(shortener models.URLShortener, batch models.BatchURLShortener, baseURL string) *ShortenHandler {
	return &ShortenHandler{shortener, batch, baseURL}
}

func NewRedirectHandler(redirector models.URLGetter, fetcher models.URLFetcher, baseURL string) *RedirectHandler {
	return &RedirectHandler{redirector, fetcher, baseURL}
}

func NewUserURLsHandler(fetcher models.URLFetcher) *UserURLsHandler {
	return &UserURLsHandler{fetcher}
}

func NewDeleteHandler(deleter models.URLDeleter) *DeleteHandler {
	return &DeleteHandler{deleter}
}

func NewPingHandler(pinger models.Pinger) *PingHandler {
	return &PingHandler{pinger}
}

func NewURLHandler(shortener models.URLShortener, batch models.BatchURLShortener, getter models.URLGetter, fetcher models.URLFetcher, deleter models.URLDeleter, pinger models.Pinger, baseURL string) *URLHandler {
	return &URLHandler{
		shorten:  NewShortenHandler(shortener, batch, baseURL),
		redirect: NewRedirectHandler(getter, fetcher, baseURL),
		userURLs: NewUserURLsHandler(fetcher),
		delete:   NewDeleteHandler(deleter),
		ping:     NewPingHandler(pinger),
	}
}

func (h *ShortenHandler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	if _, err := url.ParseRequestURI(originalURL); err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	result, err := h.shortener.ShortenURL(ctx, originalURL, userID)
	if err != nil {
		cleanErr := strings.TrimSpace(err.Error())
		http.Error(w, cleanErr, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if result.IsNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusConflict)
	}
	io.WriteString(w, result.ShortURL)
}

func (h *ShortenHandler) HandleShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	if r.Body == nil {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	var req models.ShortenRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
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
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid URL format"})
		return
	}

	result, err := h.shortener.ShortenURL(ctx, req.URL, userID)
	if err != nil {
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
	ctx := r.Context()

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	if r.Body == nil {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	var req []models.BatchShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid URL format"})
			return
		}
	}

	resp, err := h.batch.ShortenBatch(ctx, req, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to shorten batch"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *RedirectHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	id := vars["id"]

	originalURL, found := h.redirector.Get(ctx, id)
	if !found {
		http.Error(w, "Gone", http.StatusGone)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *UserURLsHandler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	urls, err := h.fetcher.GetURLsByUserID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user URLs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(urls)
}

func (h *DeleteHandler) HandleDeleteURLs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var shortIDs []string
	if err := json.NewDecoder(r.Body).Decode(&shortIDs); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(shortIDs) == 0 {
		http.Error(w, "Empty list of URLs", http.StatusBadRequest)
		return
	}

	if err := h.deleter.DeleteURLs(ctx, shortIDs, userID); err != nil {
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
