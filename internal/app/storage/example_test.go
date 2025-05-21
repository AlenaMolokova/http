package storage_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/internal/app/storage"
)

// Example_initializeStorage demonstrates initialization of different storage types.
func Example_initializeStorage() {
	// Инициализация хранилища в памяти (самый простой способ)
	memStorage, err := storage.NewStorage("", "")
	if err != nil {
		log.Fatalf("Не удалось создать хранилище в памяти: %v", err)
	}
	fmt.Println("Хранилище в памяти создано")

	// Инициализация файлового хранилища
	tempFile := "./temp_storage.json"
	_, err = storage.NewStorage("", tempFile)
	if err != nil {
		log.Fatalf("Не удалось создать файловое хранилище: %v", err)
	}
	fmt.Println("Файловое хранилище создано")

	// Инициализация хранилища PostgreSQL
	// В реальном коде используйте env переменные для хранения DSN
	dbDSN := "postgres://username:password@localhost:5432/shortener"
	_, err = storage.NewStorage(dbDSN, "")
	if err != nil {
		fmt.Printf("Предупреждение: не удалось подключиться к БД PostgreSQL: %v\n", err)
		fmt.Println("Будет использовано хранилище в памяти")
	} else {
		fmt.Println("Хранилище PostgreSQL создано")
	}

	// Проверка соединения с хранилищем
	err = memStorage.AsPinger().Ping(context.Background())
	if err != nil {
		fmt.Printf("Не удалось выполнить ping: %v\n", err)
	} else {
		fmt.Println("Ping успешен")
	}

	// Очистка тестового файла
	_ = os.Remove(tempFile)

	// Output:
	// Хранилище в памяти создано
	// Файловое хранилище создано
	// Ping успешен
}

// Example_saveAndGetURL demonstrates how to save and retrieve a URL.
func Example_saveAndGetURL() {
	// Инициализация хранилища в памяти для примера
	store, _ := storage.NewStorage("", "")

	// Данные для сохранения
	originalURL := "https://example.com/very/long/url/that/needs/shortening"
	userID := "user123"
	shortID := "abcdefg" // Пример короткого ID, в реальном коде будет генерироваться

	// Сохранение URL
	urlSaver := store.AsURLSaver()
	err := urlSaver.Save(context.Background(), shortID, originalURL, userID)
	if err != nil {
		log.Fatalf("Не удалось сохранить URL: %v", err)
	}
	fmt.Printf("URL сохранен с ID: %s\n", shortID)

	// Получение URL по короткому ID
	urlGetter := store.AsURLGetter()
	originalURLFound, exists := urlGetter.Get(context.Background(), shortID)
	if !exists {
		log.Fatalf("Не удалось найти URL по ID %s", shortID)
	}
	fmt.Printf("Получен оригинальный URL: %s\n", originalURLFound)

	// Проверяем, что это тот же URL, что мы сохраняли
	if originalURLFound == originalURL {
		fmt.Println("URL успешно получен")
	}

	// Проверка поиска по оригинальному URL
	foundID, err := urlSaver.FindByOriginalURL(context.Background(), originalURL)
	if err != nil {
		log.Fatalf("Не удалось найти URL по оригинальному адресу: %v", err)
	}
	if foundID == shortID {
		fmt.Println("URL успешно найден по оригинальному адресу")
	}

	// Output:
	// URL сохранен с ID: abcdefg
	// Получен оригинальный URL: https://example.com/very/long/url/that/needs/shortening
	// URL успешно получен
	// URL успешно найден по оригинальному адресу
}

// Example_batchSaveURL demonstrates batch saving of multiple URLs.
func Example_batchSaveURL() {
	// Инициализация хранилища
	store, _ := storage.NewStorage("", "")

	// Подготовка данных для пакетного сохранения
	userID := "user123"
	batchRequests := []models.BatchShortenRequest{
		{CorrelationID: "1", OriginalURL: "https://example.com/page1"},
		{CorrelationID: "2", OriginalURL: "https://example.com/page2"},
		{CorrelationID: "3", OriginalURL: "https://example.com/page3"},
	}

	// Вместо прямого доступа к полю impl, приводим хранилище к нужному интерфейсу
	// Добавим новый метод в Storage для доступа к BatchURLShortener
	batchShortener := store.AsBatchURLShortener()

	// Пакетное сокращение URL
	results, err := batchShortener.ShortenBatch(context.Background(), batchRequests, userID)
	if err != nil {
		log.Fatalf("Не удалось выполнить пакетное сокращение: %v", err)
	}

	// Вывод результатов
	fmt.Printf("Сокращено URL: %d\n", len(results))

	// Соответствие correlationID и shortURL
	shortURLs := make(map[string]string)
	for _, result := range results {
		fmt.Printf("Короткий URL: %s, CorrelationID: %s\n", result.ShortURL, result.CorrelationID)
		shortURLs[result.CorrelationID] = result.ShortURL
	}

	// Получение и проверка сохраненных URL
	urlGetter := store.AsURLGetter()
	for corrID, shortURL := range shortURLs {
		originalURL, exists := urlGetter.Get(context.Background(), shortURL)
		if !exists {
			log.Fatalf("Не удалось получить URL с ID %s", shortURL)
		}
		fmt.Printf("CorrelationID %s: получен URL: %s\n", corrID, strings.Split(originalURL, "/")[3])
	}

	// Output:
	// Сокращено URL: 3
	// Короткий URL: xxxxxxx, CorrelationID: 1
	// Короткий URL: xxxxxxx, CorrelationID: 2
	// Короткий URL: xxxxxxx, CorrelationID: 3
	// CorrelationID 1: получен URL: page1
	// CorrelationID 2: получен URL: page2
	// CorrelationID 3: получен URL: page3
}

// Example_fetchUserURLs demonstrates fetching all URLs belonging to a user.
func Example_fetchUserURLs() {
	// Инициализация хранилища
	store, _ := storage.NewStorage("", "")

	// Сохраняем несколько URL для пользователя
	userID := "user456"
	urlSaver := store.AsURLSaver()

	// Сохраняем три URL для тестирования
	err := urlSaver.Save(context.Background(), "shortid1", "https://example.com/user/profile", userID)
	if err != nil {
		log.Fatalf("Не удалось сохранить URL: %v", err)
	}

	err = urlSaver.Save(context.Background(), "shortid2", "https://example.com/user/settings", userID)
	if err != nil {
		log.Fatalf("Не удалось сохранить URL: %v", err)
	}

	err = urlSaver.Save(context.Background(), "shortid3", "https://example.com/user/dashboard", userID)
	if err != nil {
		log.Fatalf("Не удалось сохранить URL: %v", err)
	}

	// Получаем все URL пользователя
	urlFetcher := store.AsURLFetcher()
	userURLs, err := urlFetcher.GetURLsByUserID(context.Background(), userID)
	if err != nil {
		log.Fatalf("Не удалось получить URL пользователя: %v", err)
	}

	fmt.Printf("Получено URL пользователя: %d\n", len(userURLs))

	// Проверяем, что все наши URL присутствуют в результате
	pageTypes := map[string]bool{
		"profile":   false,
		"settings":  false,
		"dashboard": false,
	}

	for _, url := range userURLs {
		parts := strings.Split(url.OriginalURL, "/")
		if len(parts) >= 4 {
			pageType := parts[len(parts)-1]
			if _, ok := pageTypes[pageType]; ok {
				pageTypes[pageType] = true
				fmt.Printf("Найден URL: %s\n", pageType)
			}
		}
	}

	// Output:
	// Получено URL пользователя: 3
	// Найден URL: profile
	// Найден URL: settings
	// Найден URL: dashboard
}

// Example_deleteURLs demonstrates deletion of URLs.
func Example_deleteURLs() {
	// Инициализация хранилища
	store, _ := storage.NewStorage("", "")

	// Сохраняем несколько URL для пользователя
	userID := "user789"
	urlSaver := store.AsURLSaver()

	// Сохраняем URL для тестирования удаления
	err := urlSaver.Save(context.Background(), "shortid1", "https://example.com/article1", userID)
	if err != nil {
		log.Fatalf("Не удалось сохранить URL: %v", err)
	}

	err = urlSaver.Save(context.Background(), "shortid2", "https://example.com/article2", userID)
	if err != nil {
		log.Fatalf("Не удалось сохранить URL: %v", err)
	}

	err = urlSaver.Save(context.Background(), "shortid3", "https://example.com/article3", userID)
	if err != nil {
		log.Fatalf("Не удалось сохранить URL: %v", err)
	}

	// URL для удаления
	urlsToDelete := []string{"shortid1", "shortid3"}

	// Удаление URL
	urlDeleter := store.AsURLDeleter()
	err = urlDeleter.DeleteURLs(context.Background(), urlsToDelete, userID)
	if err != nil {
		log.Fatalf("Не удалось удалить URL: %v", err)
	}
	fmt.Println("URL-адреса отправлены на удаление")

	// Проверяем статус URL после удаления
	urlGetter := store.AsURLGetter()

	// Проверяем первый URL (должен быть помечен как удаленный или недоступен)
	originalURL1, exists1 := urlGetter.Get(context.Background(), "shortid1")
	if !exists1 {
		fmt.Printf("URL %s не найден после удаления\n", "shortid1")
	} else {
		fmt.Printf("URL %s все еще доступен: %s\n", "shortid1", originalURL1)
	}

	// Проверяем второй URL (не должен быть удален)
	originalURL2, exists2 := urlGetter.Get(context.Background(), "shortid2")
	if !exists2 {
		log.Fatalf("URL %s не найден, хотя не должен был быть удален", "shortid2")
	} else {
		fmt.Printf("URL %s доступен как ожидалось: %s\n", "shortid2", originalURL2)
	}

	// Output:
	// URL-адреса отправлены на удаление
	// URL shortid1 не найден после удаления
	// URL shortid2 доступен как ожидалось: https://example.com/article2
}

func TestDummy(t *testing.T) {
	// Это пустой тест, чтобы go test не жаловался на отсутствие тестов
	// Примеры выше используются как документация и запускаются через Example*
}
