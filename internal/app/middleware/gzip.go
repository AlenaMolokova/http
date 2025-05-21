package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// gzipReader реализует интерфейс io.ReadCloser для обработки gzip-сжатых данных.
// Предоставляет функциональность для чтения и корректного закрытия сжатых потоков данных.
type gzipReader struct {
	r  io.ReadCloser // Исходный поток данных
	gz *gzip.Reader  // Gzip-декомпрессор
}

// Read читает и распаковывает данные из gzip-сжатого потока.
// Реализует метод Read интерфейса io.Reader.
//
// Параметры:
//   - p []byte: буфер для чтения данных
//
// Возвращает:
//   - n int: количество прочитанных байт
//   - err error: ошибка чтения или io.EOF при достижении конца потока
func (g *gzipReader) Read(p []byte) (n int, err error) {
	return g.gz.Read(p)
}

// Close закрывает gzip-декомпрессор и исходный поток данных.
// Реализует метод Close интерфейса io.Closer.
//
// Возвращает:
//   - error: ошибка закрытия потоков данных
func (g *gzipReader) Close() error {
	if err := g.gz.Close(); err != nil {
		g.r.Close()
		return err
	}
	return g.r.Close()
}

// gzipWriter оборачивает http.ResponseWriter для автоматического сжатия ответов.
type gzipWriter struct {
	http.ResponseWriter         // Встроенный ResponseWriter
	w              *gzip.Writer // Gzip-компрессор
}

// Write сжимает данные с помощью gzip и записывает их в ответ.
// Реализует метод Write интерфейса io.Writer.
//
// Параметры:
//   - p []byte: данные для записи
//
// Возвращает:
//   - int: количество записанных байт
//   - error: ошибка записи
func (g *gzipWriter) Write(p []byte) (int, error) {
	return g.w.Write(p)
}

// GzipMiddleware создает middleware для автоматической обработки gzip-сжатия в HTTP-запросах и ответах.
// Обрабатывает запросы со сжатыми данными (Content-Encoding: gzip) и 
// сжимает ответы для клиентов, поддерживающих gzip (Accept-Encoding: gzip).
//
// Параметры:
//   - next http.Handler: следующий обработчик в цепочке middleware
//
// Возвращает:
//   - http.Handler: обработчик с поддержкой gzip-сжатия
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")

		if sendsGzip {
			body := r.Body

			gz, err := gzip.NewReader(body)
			if err != nil {
				logrus.WithError(err).Error("Failed to create gzip reader")
				http.Error(w, "Invalid gzip data", http.StatusBadRequest)
				return
			}

			r.Body = &gzipReader{
				r:  body,
				gz: gz,
			}

			if r.Header.Get("Content-Type") == "application/x-gzip" {
				r.Header.Set("Content-Type", "text/plain")
			}

			r.Header.Del("Content-Encoding")
		}

		contentType := w.Header().Get("Content-Type")
		shouldCompress := acceptsGzip && (contentType == "" ||
			strings.Contains(contentType, "application/json") ||
			strings.Contains(contentType, "text/html") ||
			strings.Contains(contentType, "text/plain"))

		if shouldCompress {
			gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				logrus.WithError(err).Error("Failed to create gzip writer")
				next.ServeHTTP(w, r)
				return
			}
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Add("Vary", "Accept-Encoding")

			gzw := &gzipWriter{
				ResponseWriter: w,
				w:              gz,
			}
			next.ServeHTTP(gzw, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}