package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/auth"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/gorilla/mux"
)

// setCookieForAuth устанавливает правильные cookie для аутентификации
func setCookieForAuth(req *http.Request, userID string) {
	signature := auth.SignData(userID)

	req.AddCookie(&http.Cookie{
		Name:  fmt.Sprintf("%s_%s", auth.CookieName, auth.CookiePartID),
		Value: userID,
	})
	req.AddCookie(&http.Cookie{
		Name:  fmt.Sprintf("%s_%s", auth.CookieName, auth.CookiePartSign),
		Value: signature,
	})
	req.AddCookie(&http.Cookie{
		Name:  auth.CookieName,
		Value: "1",
	})
}

// Benchmark для HandleShortenURL с различными сценариями
func BenchmarkHandleShortenURL(b *testing.B) {
	userID := "test-user"
	storage := memory.NewMemoryStorage()
	generator := generator.New(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	testCases := []struct {
		name string
		url  string
	}{
		{"Short_URL", "https://ya.ru"},
		{"Medium_URL", "https://example.com/path/to/resource"},
		{"Long_URL", "https://very-long-domain-name.example.com/very/long/path/to/some/resource?param1=value1&param2=value2&param3=value3"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Создаем новый запрос для каждой итерации
				body := bytes.NewBufferString(tc.url)
				req, err := http.NewRequestWithContext(context.Background(), "POST", "/shorten", body)
				if err != nil {
					b.Fatal(err)
				}
				req.Header.Set("Content-Type", "text/plain")

				// Устанавливаем правильные cookie для аутентификации
				setCookieForAuth(req, userID)

				rr := httptest.NewRecorder()
				h.ShortenHandler.HandleShortenURL(rr, req)

				// Проверяем успешность запроса
				if rr.Code != http.StatusCreated && rr.Code != http.StatusConflict {
					b.Fatalf("Expected status 201 or 409, got %d", rr.Code)
				}

				// Очищаем ресурсы
				if result := rr.Result(); result != nil {
					if closeErr := result.Body.Close(); closeErr != nil {
						b.Logf("Failed to close response body: %v", closeErr)
					}
				}
			}
		})
	}
}

// Benchmark для HandleRedirect с предварительно сохраненными URL
func BenchmarkHandleRedirect(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.New(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	ctx := context.Background()

	// Подготавливаем тестовые данные
	testURLs := []struct {
		shortID string
		longURL string
	}{
		{"short1", "https://example.com"},
		{"short2", "https://google.com"},
		{"short3", "https://github.com"},
	}

	for _, testURL := range testURLs {
		if err := storage.Save(ctx, testURL.shortID, testURL.longURL, "user123"); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Выбираем случайный URL для редиректа
		testURL := testURLs[i%len(testURLs)]

		req, err := http.NewRequestWithContext(ctx, "GET", "/"+testURL.shortID, nil)
		if err != nil {
			b.Fatal(err)
		}

		req = mux.SetURLVars(req, map[string]string{"id": testURL.shortID})

		rr := httptest.NewRecorder()

		h.RedirectHandler.HandleRedirect(rr, req)

		// Проверяем успешность запроса
		if rr.Code != http.StatusTemporaryRedirect {
			b.Fatalf("Expected status 307, got %d", rr.Code)
		}

		// Очищаем ресурсы
		if result := rr.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close response body: %v", closeErr)
			}
		}
	}
}

// Benchmark для HandleShortenURLJSON с различными размерами данных
func BenchmarkHandleShortenURLJSON(b *testing.B) {
	userID := "test-user"
	storage := memory.NewMemoryStorage()
	generator := generator.New(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	testCases := []struct {
		name string
		json string
	}{
		{"Simple_JSON", `{"url":"https://example.com"}`},
		{"Long_URL_JSON", `{"url":"https://very-long-domain-name.example.com/very/long/path/to/some/resource?param1=value1&param2=value2&param3=value3"}`},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				body := bytes.NewBufferString(tc.json)
				req, err := http.NewRequestWithContext(context.Background(), "POST", "/api/shorten", body)
				if err != nil {
					b.Fatal(err)
				}

				req.Header.Set("Content-Type", "application/json")

				// Устанавливаем правильные cookie для аутентификации
				setCookieForAuth(req, userID)

				rr := httptest.NewRecorder()
				h.ShortenHandler.HandleShortenURLJSON(rr, req)

				// Проверяем успешность запроса
				if rr.Code != http.StatusCreated && rr.Code != http.StatusConflict {
					b.Fatalf("Expected status 201 or 409, got %d", rr.Code)
				}

				// Очищаем ресурсы
				if result := rr.Result(); result != nil {
					if closeErr := result.Body.Close(); closeErr != nil {
						b.Logf("Failed to close response body: %v", closeErr)
					}
				}
			}
		})
	}
}

// Benchmark для HandleBatchShortenURL с различными размерами батчей
func BenchmarkHandleBatchShortenURL(b *testing.B) {
	userID := "test-user"
	storage := memory.NewMemoryStorage()
	generator := generator.New(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	// Создаем батчи разных размеров
	createBatch := func(size int) string {
		var batch strings.Builder
		batch.WriteString("[")
		for i := 0; i < size; i++ {
			if i > 0 {
				batch.WriteString(",")
			}
			batch.WriteString(fmt.Sprintf(`{"correlation_id":"id%d","original_url":"https://example%d.com"}`, i, i))
		}
		batch.WriteString("]")
		return batch.String()
	}

	testCases := []struct {
		name      string
		batchSize int
	}{
		{"Small_Batch_(5)", 5},
		{"Medium_Batch_(20)", 20},
		{"Large_Batch_(100)", 100},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			batchJSON := createBatch(tc.batchSize)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				body := bytes.NewBufferString(batchJSON)

				req, err := http.NewRequestWithContext(context.Background(), "POST", "/api/shorten/batch", body)
				if err != nil {
					b.Fatal(err)
				}

				req.Header.Set("Content-Type", "application/json")

				// Устанавливаем правильные cookie для аутентификации
				setCookieForAuth(req, userID)

				rr := httptest.NewRecorder()
				h.ShortenHandler.HandleBatchShortenURL(rr, req)

				// Проверяем успешность запроса
				if rr.Code != http.StatusCreated {
					b.Fatalf("Expected status 201, got %d", rr.Code)
				}

				// Очищаем ресурсы
				if result := rr.Result(); result != nil {
					if closeErr := result.Body.Close(); closeErr != nil {
						b.Logf("Failed to close response body: %v", closeErr)
					}
				}
			}
		})
	}
}

// Benchmark для HandleDeleteURLs с различными размерами списков удаления
func BenchmarkHandleDeleteURLs(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.New(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewDeleteHandler(s)

	userID := "test-user"
	ctx := context.Background()

	// Предварительно создаем URL для удаления
	var shortIDs []string
	for i := 0; i < 100; i++ {
		shortID := generator.Generate()
		if err := storage.Save(ctx, shortID, fmt.Sprintf("https://example%d.com", i), userID); err != nil {
			b.Fatal(err)
		}
		shortIDs = append(shortIDs, shortID)
	}

	testCases := []struct {
		name  string
		count int
	}{
		{"Delete_1_URL", 1},
		{"Delete_5_URLs", 5},
		{"Delete_20_URLs", 20},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Выбираем подмножество ID для удаления
				startIdx := i % len(shortIDs)
				endIdx := startIdx + tc.count
				if endIdx > len(shortIDs) {
					endIdx = len(shortIDs)
				}
				idsToDelete := shortIDs[startIdx:endIdx]

				if len(idsToDelete) < tc.count {
					idsToDelete = shortIDs[:tc.count]
				}

				// Создаем JSON с ID для удаления
				var jsonBuilder strings.Builder
				jsonBuilder.WriteString("[")
				for j, id := range idsToDelete {
					if j > 0 {
						jsonBuilder.WriteString(",")
					}
					jsonBuilder.WriteString(fmt.Sprintf(`"%s"`, id))
				}
				jsonBuilder.WriteString("]")

				body := bytes.NewBufferString(jsonBuilder.String())

				req, err := http.NewRequestWithContext(ctx, "DELETE", "/api/user/urls", body)
				if err != nil {
					b.Fatal(err)
				}

				req.Header.Set("Content-Type", "application/json")

				// Устанавливаем правильные cookie для аутентификации
				setCookieForAuth(req, userID)

				rr := httptest.NewRecorder()
				h.HandleDeleteURLs(rr, req)

				// Проверяем успешность запроса
				if rr.Code != http.StatusAccepted {
					b.Fatalf("Expected status 202, got %d", rr.Code)
				}

				// Очищаем ресурсы
				if result := rr.Result(); result != nil {
					if closeErr := result.Body.Close(); closeErr != nil {
						b.Logf("Failed to close response body: %v", closeErr)
					}
				}
			}
		})
	}
}

// Benchmark для HandleGetUserURLs с различным количеством URL у пользователя
func BenchmarkHandleGetUserURLs(b *testing.B) {
	testCases := []struct {
		name     string
		urlCount int
	}{
		{"Few_URLs_(5)", 5},
		{"Many_URLs_(50)", 50},
		{"Lots_of_URLs_(200)", 200},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			storage := memory.NewMemoryStorage()
			generator := generator.New(8)
			s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
			h := NewUserURLsHandler(s)

			userID := "test-user"
			ctx := context.Background()

			// Предварительно создаем URL для пользователя
			for i := 0; i < tc.urlCount; i++ {
				shortID := generator.Generate()
				if err := storage.Save(ctx, shortID, fmt.Sprintf("https://example%d.com", i), userID); err != nil {
					b.Fatal(err)
				}
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				req, err := http.NewRequestWithContext(ctx, "GET", "/api/user/urls", nil)
				if err != nil {
					b.Fatal(err)
				}

				// Устанавливаем правильные cookie для аутентификации
				setCookieForAuth(req, userID)

				rr := httptest.NewRecorder()
				h.HandleGetUserURLs(rr, req)

				// Проверяем успешность запроса (принимаем статус 200 и 204)
				if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
					b.Fatalf("Expected status 200 or 204, got %d", rr.Code)
				}

				// Очищаем ресурсы
				if result := rr.Result(); result != nil {
					if closeErr := result.Body.Close(); closeErr != nil {
						b.Logf("Failed to close response body: %v", closeErr)
					}
				}
			}
		})
	}
}

// Benchmark для HandlePing
func BenchmarkHandlePing(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.New(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewPingHandler(s)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req, err := http.NewRequestWithContext(context.Background(), "GET", "/ping", nil)
		if err != nil {
			b.Fatal(err)
		}

		rr := httptest.NewRecorder()
		h.HandlePing(rr, req)

		// Проверяем успешность запроса
		if rr.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", rr.Code)
		}

		// Очищаем ресурсы
		if result := rr.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close response body: %v", closeErr)
			}
		}
	}
}

// Benchmark для всего пайплайна: создание -> получение -> удаление
func BenchmarkFullPipeline(b *testing.B) {
	userID := "test-user"
	storage := memory.NewMemoryStorage()
	generator := generator.New(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	urlHandler := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()

		// 1. Создаем короткий URL
		shortenBody := bytes.NewBufferString(fmt.Sprintf("https://example%d.com", i))
		shortenReq, err := http.NewRequestWithContext(ctx, "POST", "/shorten", shortenBody)
		if err != nil {
			b.Fatal(err)
		}
		shortenReq.Header.Set("Content-Type", "text/plain")

		// Устанавливаем правильные cookie для аутентификации
		setCookieForAuth(shortenReq, userID)

		shortenRR := httptest.NewRecorder()
		urlHandler.ShortenHandler.HandleShortenURL(shortenRR, shortenReq)

		if shortenRR.Code != http.StatusCreated && shortenRR.Code != http.StatusConflict {
			b.Fatalf("Shorten failed with status %d", shortenRR.Code)
		}

		// 2. Получаем список URL пользователя
		getUserReq, err := http.NewRequestWithContext(ctx, "GET", "/api/user/urls", nil)
		if err != nil {
			b.Fatal(err)
		}

		// Устанавливаем правильные cookie для аутентификации
		setCookieForAuth(getUserReq, userID)

		getUserRR := httptest.NewRecorder()
		urlHandler.UserURLsHandler.HandleGetUserURLs(getUserRR, getUserReq)

		// Очищаем ресурсы
		if result := shortenRR.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close shorten response body: %v", closeErr)
			}
		}

		if result := getUserRR.Result(); result != nil {
			if closeErr := result.Body.Close(); closeErr != nil {
				b.Logf("Failed to close get user response body: %v", closeErr)
			}
		}
	}
}

// Benchmark для параллельных запросов
func BenchmarkConcurrentShortenURL(b *testing.B) {
	userID := "test-user"
	storage := memory.NewMemoryStorage()
	generator := generator.New(8)
	s := service.NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	h := NewURLHandler(s, s, s, s, s, s, "http://localhost:8080")

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			body := bytes.NewBufferString(fmt.Sprintf("https://concurrent-example%d.com", counter))
			req, err := http.NewRequestWithContext(context.Background(), "POST", "/shorten", body)
			if err != nil {
				b.Fatal(err)
			}
			req.Header.Set("Content-Type", "text/plain")

			// Устанавливаем правильные cookie для аутентификации
			setCookieForAuth(req, userID)

			rr := httptest.NewRecorder()
			h.ShortenHandler.HandleShortenURL(rr, req)

			if rr.Code != http.StatusCreated && rr.Code != http.StatusConflict {
				b.Fatalf("Expected status 201 or 409, got %d", rr.Code)
			}

			// Очищаем ресурсы
			if result := rr.Result(); result != nil {
				if closeErr := result.Body.Close(); closeErr != nil {
					b.Logf("Failed to close response body: %v", closeErr)
				}
			}
			counter++
		}
	})
}
