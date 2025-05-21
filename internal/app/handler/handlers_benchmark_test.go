package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/gorilla/mux"
)

func BenchmarkHandleShortenURL(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	body := bytes.NewBufferString("https://example.com")
	req, err := http.NewRequestWithContext(context.Background(), "POST", "/shorten", body)
	if err != nil {
		b.Fatal(err)
	}
	req.Header.Set("Content-Type", "text/plain")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		h.HandleShortenURL(rr, req)
	}
}

func BenchmarkHandleRedirect(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	ctx := context.Background()
	storage.Save(ctx, "shortID", "https://example.com", "user123")
	req, err := http.NewRequestWithContext(ctx, "GET", "/shortID", nil)
	if err != nil {
		b.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{"id": "shortID"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		h.HandleRedirect(rr, req)
	}
}

func BenchmarkHandleShortenURLJSON(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	body := bytes.NewBufferString(`{"url":"https://example.com"}`)
	req, err := http.NewRequestWithContext(context.Background(), "POST", "/api/shorten", body)
	if err != nil {
		b.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		h.HandleShortenURLJSON(rr, req)
	}
}
