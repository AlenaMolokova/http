package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	size int
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status: http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int,error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size+=size
	return size, err
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		start := time.Now()
		rw := newResponseWriter(w)

		next.ServeHTTP(rw, r)

		duration :=time.Since(start)

		entry := logrus.WithFields(logrus.Fields{
			"uri": r.RequestURI,
			"method": r.Method,
			"duration": duration.String(),
			"status": rw.status,
			"response_size": rw.size,
			"content_type": r.Header.Get("Content-Type"),
		})

		if r.Method == http.MethodPost && r.RequestURI=="/" {
			entry =entry.WithField("operation", "shorten_url")
		} else if r.Method == http.MethodGet {
			entry = entry.WithField("operation", "redirect")
		}

		entry.Info("Request processed")
	})
}