package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

func acceptsGzip(r *http.Request) bool {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	return strings.Contains(acceptEncoding, "gzip")
}

func isGzipped(r *http.Request) bool {
	contentEncoding := r.Header.Get("Content-Encoding")
	return strings.Contains(contentEncoding, "gzip")
}

func canCompress(contentType string) bool {
	return strings.Contains(contentType, "application/json") || 
	       strings.Contains(contentType, "text/html") ||
	       strings.Contains(contentType, "text/plain")
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (grw gzipResponseWriter) Write(data []byte) (int, error) {
	return grw.Writer.Write(data)
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isGzipped(r) {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				logrus.WithError(err).Error("Failed to create gzip reader")
				http.Error(w, "Failed to read gzipped request", http.StatusBadRequest)
				return
			}
			defer gzipReader.Close()
			
			r.Body = gzipReader
		}

		contentType := w.Header().Get("Content-Type")
		if acceptsGzip(r) && (contentType == "" || canCompress(contentType)) {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Add("Vary", "Accept-Encoding")
			
			gzipWriter := gzip.NewWriter(w)
			defer gzipWriter.Close()
			
			next.ServeHTTP(gzipResponseWriter{ResponseWriter: w, Writer: gzipWriter}, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

type gzipReadCloser struct {
	io.ReadCloser
	gzipReader *gzip.Reader
}

func (gz *gzipReadCloser) Read(p []byte) (n int, err error) {
	return gz.gzipReader.Read(p)
}

func (gz *gzipReadCloser) Close() error {
	if err := gz.gzipReader.Close(); err != nil {
		return err
	}
	return gz.ReadCloser.Close()
}