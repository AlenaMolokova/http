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

// ShortenHandler обрабатывает запросы на сокращение URL.
type ShortenHandler struct {
	shortener models.URLShortener
	batch     models.BatchURLShortener
	baseURL   string
}

// RedirectHandler обрабатывает запросы на перенаправление по короткому URL.
type RedirectHandler struct {
	redirector models.URLGetter
	fetcher    models.URLFetcher
	baseURL    string
}

// UserURLsHandler обрабатывает запросы на получение URL, принадлежащих пользователю.
type UserURLsHandler struct {
	fetcher models.URLFetcher
}

// DeleteHandler обрабатывает запросы на удаление URL.
type DeleteHandler struct {
	deleter models.URLDeleter
}

// PingHandler обрабатывает запросы на проверку соединения с хранилищем.
type PingHandler struct {
	pinger models.Pinger
}

// URLHandler объединяет все обработчики URL и предоставляет единый интерфейс для обработки различных запросов.
type URLHandler struct {
	shorten  *ShortenHandler
	redirect *RedirectHandler
	userURLs *UserURLsHandler
	delete   *DeleteHandler
	ping     *PingHandler
}

// NewShortenHandler создает новый обработчик для сокращения URL.
//
// Параметры:
//   - shortener: сервис для сокращения отдельных URL
//   - batch: сервис для пакетного сокращения URL
//   - baseURL: базовый URL сервиса
//
// Возвращает:
//   - *ShortenHandler: новый обработчик
func NewShortenHandler(shortener models.URLShortener, batch models.BatchURLShortener, baseURL string) *ShortenHandler {
	return &ShortenHandler{shortener, batch, baseURL}
}

// NewRedirectHandler создает новый обработчик для перенаправления по коротким URL.
//
// Параметры:
//   - redirector: сервис для получения оригинальных URL
//   - fetcher: сервис для получения URL пользователя
//   - baseURL: базовый URL сервиса
//
// Возвращает:
//   - *RedirectHandler: новый обработчик
func NewRedirectHandler(redirector models.URLGetter, fetcher models.URLFetcher, baseURL string) *RedirectHandler {
	return &RedirectHandler{redirector, fetcher, baseURL}
}

// NewUserURLsHandler создает новый обработчик для получения URL пользователя.
//
// Параметры:
//   - fetcher: сервис для получения URL пользователя
//
// Возвращает:
//   - *UserURLsHandler: новый обработчик
func NewUserURLsHandler(fetcher models.URLFetcher) *UserURLsHandler {
	return &UserURLsHandler{fetcher}
}

// NewDeleteHandler создает новый обработчик для удаления URL.
//
// Параметры:
//   - deleter: сервис для удаления URL
//
// Возвращает:
//   - *DeleteHandler: новый обработчик
func NewDeleteHandler(deleter models.URLDeleter) *DeleteHandler {
	return &DeleteHandler{deleter}
}

// NewPingHandler создает новый обработчик для проверки соединения с хранилищем.
//
// Параметры:
//   - pinger: сервис для проверки соединения
//
// Возвращает:
//   - *PingHandler: новый обработчик
func NewPingHandler(pinger models.Pinger) *PingHandler {
	return &PingHandler{pinger}
}

// NewURLHandler создает новый комбинированный обработчик для всех операций с URL.
//
// Параметры:
//   - shortener: сервис для сокращения URL
//   - batch: сервис для пакетного сокращения URL
//   - getter: сервис для получения оригинальных URL
//   - fetcher: сервис для получения URL пользователя
//   - deleter: сервис для удаления URL
//   - pinger: сервис для проверки соединения с хранилищем
//   - baseURL: базовый URL сервиса
//
// Возвращает:
//   - *URLHandler: новый комбинированный обработчик
func NewURLHandler(shortener models.URLShortener, batch models.BatchURLShortener, getter models.URLGetter, fetcher models.URLFetcher, deleter models.URLDeleter, pinger models.Pinger, baseURL string) *URLHandler {
	return &URLHandler{
		shorten:  NewShortenHandler(shortener, batch, baseURL),
		redirect: NewRedirectHandler(getter, fetcher, baseURL),
		userURLs: NewUserURLsHandler(fetcher),
		delete:   NewDeleteHandler(deleter),
		ping:     NewPingHandler(pinger),
	}
}

// HandleShortenURL обрабатывает запросы на сокращение URL в текстовом формате.
// Поддерживает HTTP методы POST.
// Принимает URL в теле запроса в виде текста.
// Возвращает сокращенный URL в теле ответа.
//
// Коды ответа:
//   - 201 Created: URL успешно сокращен (новый URL)
//   - 409 Conflict: URL уже был сокращен ранее
//   - 400 Bad Request: неверный формат запроса
//   - 500 Internal Server Error: внутренняя ошибка сервера
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

// HandleShortenURLJSON обрабатывает запросы на сокращение URL в формате JSON.
// Поддерживает HTTP методы POST.
// Принимает JSON-объект с полем "url" в теле запроса.
// Возвращает JSON-объект с полем "result", содержащим сокращенный URL.
//
// Коды ответа:
//   - 201 Created: URL успешно сокращен (новый URL)
//   - 409 Conflict: URL уже был сокращен ранее
//   - 400 Bad Request: неверный формат запроса
//   - 500 Internal Server Error: внутренняя ошибка сервера
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

// HandleBatchShortenURL обрабатывает запросы на пакетное сокращение URL.
// Поддерживает HTTP методы POST.
// Принимает массив JSON-объектов с полями "correlation_id" и "original_url" в теле запроса.
// Возвращает массив JSON-объектов с полями "correlation_id" и "short_url".
//
// Коды ответа:
//   - 201 Created: URLs успешно сокращены
//   - 400 Bad Request: неверный формат запроса
//   - 500 Internal Server Error: внутренняя ошибка сервера
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

// HandleRedirect обрабатывает запросы на перенаправление по короткому URL.
// Поддерживает HTTP методы GET.
// Извлекает идентификатор из URL-пути и перенаправляет на оригинальный URL.
//
// Коды ответа:
//   - 307 Temporary Redirect: успешное перенаправление
//   - 410 Gone: URL был удален или не существует
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

// HandleGetUserURLs обрабатывает запросы на получение всех URL, принадлежащих пользователю.
// Поддерживает HTTP методы GET.
// Извлекает идентификатор пользователя из cookie и возвращает список его URL.
//
// Коды ответа:
//   - 200 OK: список URL успешно получен
//   - 204 No Content: у пользователя нет сохраненных URL
//   - 500 Internal Server Error: внутренняя ошибка сервера
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

// HandleDeleteURLs обрабатывает запросы на удаление URL.
// Поддерживает HTTP методы DELETE.
// Принимает массив идентификаторов URL для удаления в теле запроса.
// Удаление выполняется асинхронно.
//
// Коды ответа:
//   - 202 Accepted: запрос на удаление принят
//   - 400 Bad Request: неверный формат запроса
//   - 401 Unauthorized: пользователь не авторизован
//   - 500 Internal Server Error: внутренняя ошибка сервера
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

// HandlePing обрабатывает запросы на проверку соединения с хранилищем данных.
// Поддерживает HTTP методы GET.
// Проверяет доступность базы данных.
//
// Коды ответа:
//   - 200 OK: соединение с базой данных установлено или хранилище не требует проверки соединения
//   - 500 Internal Server Error: ошибка соединения с базой данных
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

// HandleShortenURL делегирует обработку запроса на сокращение URL в текстовом формате соответствующему обработчику.
func (h *URLHandler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	h.shorten.HandleShortenURL(w, r)
}

// HandleShortenURLJSON делегирует обработку запроса на сокращение URL в формате JSON соответствующему обработчику.
func (h *URLHandler) HandleShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	h.shorten.HandleShortenURLJSON(w, r)
}

// HandleBatchShortenURL делегирует обработку запроса на пакетное сокращение URL соответствующему обработчику.
func (h *URLHandler) HandleBatchShortenURL(w http.ResponseWriter, r *http.Request) {
	h.shorten.HandleBatchShortenURL(w, r)
}

// HandleRedirect делегирует обработку запроса на перенаправление по короткому URL соответствующему обработчику.
func (h *URLHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	h.redirect.HandleRedirect(w, r)
}

// HandleGetUserURLs делегирует обработку запроса на получение URL пользователя соответствующему обработчику.
func (h *URLHandler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	h.userURLs.HandleGetUserURLs(w, r)
}

// HandleDeleteURLs делегирует обработку запроса на удаление URL соответствующему обработчику.
func (h *URLHandler) HandleDeleteURLs(w http.ResponseWriter, r *http.Request) {
	h.delete.HandleDeleteURLs(w, r)
}

// HandlePing делегирует обработку запроса на проверку соединения с хранилищем данных соответствующему обработчику.
func (h *URLHandler) HandlePing(w http.ResponseWriter, r *http.Request) {
	h.ping.HandlePing(w, r)
}