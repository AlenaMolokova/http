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
		h.ShortenHandler.HandleShortenURL(rr, req)

		// Очищаем ресурсы после каждой итерации
		if result := rr.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close response body: %v", closeErr)
			}
		}
	}
}

func BenchmarkHandleRedirect(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	ctx := context.Background()
	if err := storage.Save(ctx, "shortID", "https://example.com", "user123"); err != nil {
		b.Fatal(err)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "/shortID", nil)
	if err != nil {
		b.Fatal(err)
	}
	req = mux.SetURLVars(req, map[string]string{"id": "shortID"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		h.RedirectHandler.HandleRedirect(rr, req)

		// Очищаем ресурсы после каждой итерации
		if result := rr.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close response body: %v", closeErr)
			}
		}
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
		h.ShortenHandler.HandleShortenURLJSON(rr, req)

		// Очищаем ресурсы после каждой итерации
		if result := rr.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close response body: %v", closeErr)
			}
		}
	}
}

func BenchmarkHandleDeleteURLs(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewDeleteHandler(s)

	userID := "test-user"
	shortID := generator.Generate()
	if err := storage.Save(context.Background(), shortID, "https://example.com", userID); err != nil {
		b.Fatal(err)
	}

	body := bytes.NewBufferString(`["` + shortID + `"]`)
	req, err := http.NewRequestWithContext(context.Background(), "DELETE", "/api/user/urls", body)
	if err != nil {
		b.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "user_id", Value: userID})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		h.HandleDeleteURLs(rr, req)

		if result := rr.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close response body: %v", closeErr)
			}
		}
	}
}

func BenchmarkHandleGetUserURLs(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewUserURLsHandler(s)

	userID := "test-user"
	shortID := generator.Generate()
	if err := storage.Save(context.Background(), shortID, "https://example.com", userID); err != nil {
		b.Fatal(err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/api/user/urls", nil)
	if err != nil {
		b.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: "user_id", Value: userID})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		h.HandleGetUserURLs(rr, req)

		if result := rr.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close response body: %v", closeErr)
			}
		}
	}
}

func BenchmarkHandlePing(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewPingHandler(s)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/ping", nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		h.HandlePing(rr, req)

		if result := rr.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close response body: %v", closeErr)
			}
		}
	}
}
