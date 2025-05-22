package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// responseWriter оборачивает стандартный http.ResponseWriter,
// добавляя возможность отслеживания статус-кода ответа и его размера.
type responseWriter struct {
	http.ResponseWriter      // Встроенный ResponseWriter
	status              int  // Код HTTP-статуса ответа
	size                int  // Размер ответа в байтах
	wroteHeader         bool // Флаг, указывающий был ли записан заголовок
}

// newResponseWriter создает новый экземпляр responseWriter, оборачивающий стандартный http.ResponseWriter.
// По умолчанию устанавливает статус ответа 200 OK.
//
// Параметры:
//   - w http.ResponseWriter: стандартный ResponseWriter для обертывания
//
// Возвращает:
//   - *responseWriter: новый объект responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

// WriteHeader устанавливает код HTTP-статуса для ответа.
// Если заголовок уже был записан, повторный вызов игнорируется.
//
// Параметры:
//   - code int: код HTTP-статуса
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write записывает данные в ответ и подсчитывает их размер.
// Если заголовок еще не был записан, автоматически устанавливает код статуса 200 OK.
//
// Параметры:
//   - b []byte: данные для записи
//
// Возвращает:
//   - int: количество записанных байт
//   - error: ошибка записи
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// LoggingMiddleware создает middleware для логирования HTTP-запросов.
// Фиксирует информацию о запросе, включая URI, метод, длительность обработки,
// код статуса ответа, размер ответа и тип контента.
// Для определенных типов запросов (POST "/" и GET) добавляет метку операции.
//
// Параметры:
//   - next http.Handler: следующий обработчик в цепочке middleware
//
// Возвращает:
//   - http.Handler: обработчик с функциональностью логирования
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		entry := logrus.WithFields(logrus.Fields{
			"uri":           r.RequestURI,
			"method":        r.Method,
			"duration":      duration.String(),
			"status":        rw.status,
			"response_size": rw.size,
			"content_type":  r.Header.Get("Content-Type"),
		})

		if r.Method == http.MethodPost && r.RequestURI == "/" {
			entry = entry.WithField("operation", "shorten_url")
		} else if r.Method == http.MethodGet {
			entry = entry.WithField("operation", "redirect")
		}

		entry.Info("Request processed")
	})
}
