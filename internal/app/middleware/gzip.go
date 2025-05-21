package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type gzipReader struct {
	r  io.ReadCloser
	gz *gzip.Reader
}

func (g *gzipReader) Read(p []byte) (n int, err error) {
	return g.gz.Read(p)
}

func (g *gzipReader) Close() error {
	if err := g.gz.Close(); err != nil {
		g.r.Close()
		return err
	}
	return g.r.Close()
}

type gzipWriter struct {
	http.ResponseWriter
	w *gzip.Writer
}

func (g *gzipWriter) Write(p []byte) (int, error) {
	return g.w.Write(p)
}


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