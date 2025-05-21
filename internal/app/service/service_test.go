package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/models"
)

type MockURLSaver struct {
	SaveFunc              func(ctx context.Context, shortID, originalURL, userID string) error
	FindByOriginalURLFunc func(ctx context.Context, originalURL string) (string, error)
}

func (m *MockURLSaver) Save(ctx context.Context, shortID, originalURL, userID string) error {
	return m.SaveFunc(ctx, shortID, originalURL, userID)
}

func (m *MockURLSaver) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	return m.FindByOriginalURLFunc(ctx, originalURL)
}

type MockURLBatchSaver struct {
	SaveBatchFunc func(ctx context.Context, items map[string]string, userID string) error
}

func (m *MockURLBatchSaver) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	return m.SaveBatchFunc(ctx, items, userID)
}

type MockURLGetter struct {
	GetFunc func(ctx context.Context, shortID string) (string, bool)
}

func (m *MockURLGetter) Get(ctx context.Context, shortID string) (string, bool) {
	return m.GetFunc(ctx, shortID)
}

type MockURLFetcher struct {
	GetURLsByUserIDFunc func(ctx context.Context, userID string) ([]models.UserURL, error)
}

func (m *MockURLFetcher) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	return m.GetURLsByUserIDFunc(ctx, userID)
}

type MockURLDeleter struct {
	DeleteURLsFunc func(ctx context.Context, shortIDs []string, userID string) error
}

func (m *MockURLDeleter) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	return m.DeleteURLsFunc(ctx, shortIDs, userID)
}

type MockPinger struct {
	PingFunc func(ctx context.Context) error
}

func (m *MockPinger) Ping(ctx context.Context) error {
	return m.PingFunc(ctx)
}

type MockGenerator struct {
	GenerateFunc func() string
}

func (m *MockGenerator) Generate() string {
	return m.GenerateFunc()
}

func TestNewService(t *testing.T) {
	saver := &MockURLSaver{}
	batch := &MockURLBatchSaver{}
	getter := &MockURLGetter{}
	fetcher := &MockURLFetcher{}
	deleter := &MockURLDeleter{}
	pinger := &MockPinger{}
	gen := &MockGenerator{}
	baseURL := "http://example.com"

	service := NewService(saver, batch, getter, fetcher, deleter, pinger, gen, baseURL)

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.BaseURL != baseURL {
		t.Errorf("Expected BaseURL to be %s, got %s", baseURL, service.BaseURL)
	}

	if service.cache == nil {
		t.Error("cache is nil")
	}
}

func TestService_ShortenURL(t *testing.T) {
	ctx := context.Background()
	originalURL := "https://example.com"
	userID := "user1"
	shortID := "abc123"
	baseURL := "http://short.url"

	tests := []struct {
		name              string
		findByOriginalURL func(ctx context.Context, originalURL string) (string, error)
		saveFunc          func(ctx context.Context, shortID, originalURL, userID string) error
		generateFunc      func() string
		expectedResult    models.ShortenResult
		expectedError     bool
		expectedErrorMsg  string
	}{
		{
			name: "URL already exists",
			findByOriginalURL: func(ctx context.Context, url string) (string, error) {
				return shortID, nil
			},
			saveFunc: func(ctx context.Context, shortID, originalURL, userID string) error {
				return nil
			},
			generateFunc: func() string {
				return shortID
			},
			expectedResult: models.ShortenResult{
				ShortURL: baseURL + "/" + shortID,
				IsNew:    false,
			},
			expectedError: false,
		},
		{
			name: "New URL success",
			findByOriginalURL: func(ctx context.Context, url string) (string, error) {
				return "", nil
			},
			saveFunc: func(ctx context.Context, sid, origURL, uid string) error {
				if sid != shortID || origURL != originalURL || uid != userID {
					t.Errorf("Save called with unexpected parameters: shortID=%s, originalURL=%s, userID=%s", sid, origURL, uid)
				}
				return nil
			},
			generateFunc: func() string {
				return shortID
			},
			expectedResult: models.ShortenResult{
				ShortURL: baseURL + "/" + shortID,
				IsNew:    true,
			},
			expectedError: false,
		},
		{
			name: "FindByOriginalURL error",
			findByOriginalURL: func(ctx context.Context, url string) (string, error) {
				return "", errors.New("database error")
			},
			saveFunc: func(ctx context.Context, shortID, originalURL, userID string) error {
				return nil
			},
			generateFunc: func() string {
				return shortID
			},
			expectedResult:   models.ShortenResult{},
			expectedError:    true,
			expectedErrorMsg: "error finding URL",
		},
		{
			name: "Generate failure",
			findByOriginalURL: func(ctx context.Context, url string) (string, error) {
				return "", nil
			},
			saveFunc: func(ctx context.Context, shortID, originalURL, userID string) error {
				return nil
			},
			generateFunc: func() string {
				return ""
			},
			expectedResult:   models.ShortenResult{},
			expectedError:    true,
			expectedErrorMsg: "failed to generate short ID",
		},
		{
			name: "Save error",
			findByOriginalURL: func(ctx context.Context, url string) (string, error) {
				return "", nil
			},
			saveFunc: func(ctx context.Context, shortID, originalURL, userID string) error {
				return errors.New("save error")
			},
			generateFunc: func() string {
				return shortID
			},
			expectedResult:   models.ShortenResult{},
			expectedError:    true,
			expectedErrorMsg: "error saving URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &MockURLSaver{
				SaveFunc:              tt.saveFunc,
				FindByOriginalURLFunc: tt.findByOriginalURL,
			}
			generator := &MockGenerator{
				GenerateFunc: tt.generateFunc,
			}

			service := NewService(
				saver,
				&MockURLBatchSaver{},
				&MockURLGetter{},
				&MockURLFetcher{},
				&MockURLDeleter{},
				&MockPinger{},
				generator,
				baseURL,
			)

			result, err := service.ShortenURL(ctx, originalURL, userID)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedErrorMsg)
				} else if tt.expectedErrorMsg != "" && !errors.Is(err, errors.New(tt.expectedErrorMsg)) {
					errMsg := err.Error()
					if !errorContains(errMsg, tt.expectedErrorMsg) {
						t.Errorf("Error does not contain '%s': %v", tt.expectedErrorMsg, err)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.ShortURL != tt.expectedResult.ShortURL {
				t.Errorf("ShortURL: expected %s, got %s", tt.expectedResult.ShortURL, result.ShortURL)
			}

			if result.IsNew != tt.expectedResult.IsNew {
				t.Errorf("IsNew: expected %v, got %v", tt.expectedResult.IsNew, result.IsNew)
			}
		})
	}
}

func errorContains(actual, expected string) bool {
	return errors.Is(errors.New(actual), errors.New(expected)) ||
		errors.Is(errors.New(expected), errors.New(actual)) ||
		actual == expected ||
		len(actual) >= len(expected) &&
			actual[:len(expected)] == expected
}

func TestService_ShortenBatch(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	baseURL := "http://short.url"
	shortID1 := "abc123"
	shortID2 := "def456"

	generateCalls := 0
	generateFunc := func() string {
		generateCalls++
		if generateCalls == 1 {
			return shortID1
		}
		return shortID2
	}

	tests := []struct {
		name            string
		items           []models.BatchShortenRequest
		saveBatchFunc   func(ctx context.Context, items map[string]string, userID string) error
		expectedResults []models.BatchShortenResponse
		expectedError   bool
	}{
		{
			name: "Successful batch shorten",
			items: []models.BatchShortenRequest{
				{
					CorrelationID: "1",
					OriginalURL:   "https://example1.com",
				},
				{
					CorrelationID: "2",
					OriginalURL:   "https://example2.com",
				},
			},
			saveBatchFunc: func(ctx context.Context, items map[string]string, uid string) error {
				if uid != userID {
					t.Errorf("SaveBatch called with unexpected userID: %s", uid)
				}
				if len(items) != 2 {
					t.Errorf("SaveBatch called with unexpected number of items: %d", len(items))
				}
				if items[shortID1] != "https://example1.com" {
					t.Errorf("SaveBatch items[%s] = %s, expected %s", shortID1, items[shortID1], "https://example1.com")
				}
				if items[shortID2] != "https://example2.com" {
					t.Errorf("SaveBatch items[%s] = %s, expected %s", shortID2, items[shortID2], "https://example2.com")
				}
				return nil
			},
			expectedResults: []models.BatchShortenResponse{
				{
					CorrelationID: "1",
					ShortURL:      baseURL + "/" + shortID1,
				},
				{
					CorrelationID: "2",
					ShortURL:      baseURL + "/" + shortID2,
				},
			},
			expectedError: false,
		},
		{
			name: "SaveBatch error",
			items: []models.BatchShortenRequest{
				{
					CorrelationID: "1",
					OriginalURL:   "https://example1.com",
				},
			},
			saveBatchFunc: func(ctx context.Context, items map[string]string, uid string) error {
				return errors.New("batch save error")
			},
			expectedResults: nil,
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generateCalls = 0
			batchSaver := &MockURLBatchSaver{
				SaveBatchFunc: tt.saveBatchFunc,
			}
			generator := &MockGenerator{
				GenerateFunc: generateFunc,
			}

			service := NewService(
				&MockURLSaver{},
				batchSaver,
				&MockURLGetter{},
				&MockURLFetcher{},
				&MockURLDeleter{},
				&MockPinger{},
				generator,
				baseURL,
			)

			results, err := service.ShortenBatch(ctx, tt.items, userID)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(results, tt.expectedResults) {
				t.Errorf("Expected results %+v, got %+v", tt.expectedResults, results)
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	ctx := context.Background()
	shortID := "abc123"
	originalURL := "https://example.com"

	tests := []struct {
		name           string
		getFunc        func(ctx context.Context, shortID string) (string, bool)
		expectedURL    string
		expectedExists bool
	}{
		{
			name: "URL exists",
			getFunc: func(ctx context.Context, sid string) (string, bool) {
				if sid != shortID {
					t.Errorf("Get called with unexpected shortID: %s", sid)
				}
				return originalURL, true
			},
			expectedURL:    originalURL,
			expectedExists: true,
		},
		{
			name: "URL does not exist",
			getFunc: func(ctx context.Context, sid string) (string, bool) {
				return "", false
			},
			expectedURL:    "",
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter := &MockURLGetter{
				GetFunc: tt.getFunc,
			}

			service := NewService(
				&MockURLSaver{},
				&MockURLBatchSaver{},
				getter,
				&MockURLFetcher{},
				&MockURLDeleter{},
				&MockPinger{},
				&MockGenerator{},
				"http://short.url",
			)

			url, exists := service.Get(ctx, shortID)

			if url != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, url)
			}

			if exists != tt.expectedExists {
				t.Errorf("Expected exists %v, got %v", tt.expectedExists, exists)
			}
		})
	}
}

func TestService_GetURLsByUserID(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	baseURL := "http://short.url"

	userURLs := []models.UserURL{
		{
			ShortURL:    "abc123",
			OriginalURL: "https://example1.com",
			UserID:      userID,
		},
		{
			ShortURL:    "def456",
			OriginalURL: "https://example2.com",
			UserID:      userID,
		},
	}

	expectedURLs := []models.UserURL{
		{
			ShortURL:    baseURL + "/abc123",
			OriginalURL: "https://example1.com",
			UserID:      userID,
		},
		{
			ShortURL:    baseURL + "/def456",
			OriginalURL: "https://example2.com",
			UserID:      userID,
		},
	}

	tests := []struct {
		name                string
		getURLsByUserIDFunc func(ctx context.Context, userID string) ([]models.UserURL, error)
		callTwice           bool
		expectedURLs        []models.UserURL
		expectedError       bool
	}{
		{
			name: "First fetch success",
			getURLsByUserIDFunc: func(ctx context.Context, uid string) ([]models.UserURL, error) {
				if uid != userID {
					t.Errorf("GetURLsByUserID called with unexpected userID: %s", uid)
				}
				return deepCopyURLs(userURLs), nil
			},
			callTwice:     false,
			expectedURLs:  expectedURLs,
			expectedError: false,
		},
		{
			name: "Fetch from cache",
			getURLsByUserIDFunc: func(ctx context.Context, uid string) ([]models.UserURL, error) {
				if uid != userID {
					t.Errorf("GetURLsByUserID called with unexpected userID: %s", uid)
				}
				return deepCopyURLs(userURLs), nil
			},
			callTwice:     true,
			expectedURLs:  expectedURLs,
			expectedError: false,
		},
		{
			name: "Fetch error",
			getURLsByUserIDFunc: func(ctx context.Context, uid string) ([]models.UserURL, error) {
				return nil, errors.New("fetch error")
			},
			callTwice:     false,
			expectedURLs:  nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &MockURLFetcher{
				GetURLsByUserIDFunc: tt.getURLsByUserIDFunc,
			}

			service := NewService(
				&MockURLSaver{},
				&MockURLBatchSaver{},
				&MockURLGetter{},
				fetcher,
				&MockURLDeleter{},
				&MockPinger{},
				&MockGenerator{},
				baseURL,
			)

			urls, err := service.GetURLsByUserID(ctx, userID)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !urlsEqual(urls, tt.expectedURLs) {
				t.Errorf("Expected URLs %+v, got %+v", tt.expectedURLs, urls)
			}

			if tt.callTwice {
				calls := 0
				fetcher.GetURLsByUserIDFunc = func(ctx context.Context, uid string) ([]models.UserURL, error) {
					calls++
					return nil, errors.New("this should not be called")
				}

				cachedURLs, err := service.GetURLsByUserID(ctx, userID)
				if err != nil {
					t.Fatalf("Unexpected error on second call: %v", err)
				}

				if calls > 0 {
					t.Error("GetURLsByUserID was called again, expected to use cache")
				}

				if !urlsEqual(cachedURLs, tt.expectedURLs) {
					t.Errorf("Expected cached URLs %+v, got %+v", tt.expectedURLs, cachedURLs)
				}
			}
		})
	}
}

func deepCopyURLs(urls []models.UserURL) []models.UserURL {
	result := make([]models.UserURL, len(urls))
	for i, url := range urls {
		result[i] = models.UserURL{
			ShortURL:    url.ShortURL,
			OriginalURL: url.OriginalURL,
			UserID:      url.UserID,
			IsDeleted:   url.IsDeleted,
		}
	}
	return result
}

func urlsEqual(a, b []models.UserURL) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].ShortURL != b[i].ShortURL ||
			a[i].OriginalURL != b[i].OriginalURL ||
			a[i].UserID != b[i].UserID {
			return false
		}
	}

	return true
}

func TestService_DeleteURLs(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	shortIDs := []string{"abc123", "def456"}

	tests := []struct {
		name           string
		deleteURLsFunc func(ctx context.Context, shortIDs []string, userID string) error
		expectedError  bool
	}{
		{
			name: "Delete success",
			deleteURLsFunc: func(ctx context.Context, sids []string, uid string) error {
				if uid != userID {
					t.Errorf("DeleteURLs called with unexpected userID: %s", uid)
				}
				if !reflect.DeepEqual(sids, shortIDs) {
					t.Errorf("DeleteURLs called with unexpected shortIDs: %v", sids)
				}
				return nil
			},
			expectedError: false,
		},
		{
			name: "Delete error",
			deleteURLsFunc: func(ctx context.Context, sids []string, uid string) error {
				return errors.New("delete error")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleter := &MockURLDeleter{
				DeleteURLsFunc: tt.deleteURLsFunc,
			}

			fetcher := &MockURLFetcher{
				GetURLsByUserIDFunc: func(ctx context.Context, uid string) ([]models.UserURL, error) {
					return []models.UserURL{
						{
							ShortURL:    "abc123",
							OriginalURL: "https://example1.com",
							UserID:      userID,
						},
					}, nil
				},
			}

			service := NewService(
				&MockURLSaver{},
				&MockURLBatchSaver{},
				&MockURLGetter{},
				fetcher,
				deleter,
				&MockPinger{},
				&MockGenerator{},
				"http://short.url",
			)

			_, _ = service.GetURLsByUserID(ctx, userID)

			service.cacheMu.RLock()
			if len(service.cache[userID]) == 0 {
				t.Error("Cache should not be empty before DeleteURLs")
			}
			service.cacheMu.RUnlock()

			err := service.DeleteURLs(ctx, shortIDs, userID)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			service.cacheMu.RLock()
			_, exists := service.cache[userID]
			service.cacheMu.RUnlock()

			if exists {
				t.Error("Cache should be cleared after DeleteURLs")
			}
		})
	}
}

func TestService_Ping(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		pingFunc    func(ctx context.Context) error
		expectError bool
	}{
		{
			name: "Ping success",
			pingFunc: func(ctx context.Context) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "Ping error",
			pingFunc: func(ctx context.Context) error {
				return errors.New("ping error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pinger := &MockPinger{
				PingFunc: tt.pingFunc,
			}

			service := NewService(
				&MockURLSaver{},
				&MockURLBatchSaver{},
				&MockURLGetter{},
				&MockURLFetcher{},
				&MockURLDeleter{},
				pinger,
				&MockGenerator{},
				"http://short.url",
			)

			err := service.Ping(ctx)

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestService_CacheClearOnShortenURL(t *testing.T) {
	ctx := context.Background()
	userID := "user1"
	originalURL := "https://example.com"
	shortID := "abc123"

	saver := &MockURLSaver{
		SaveFunc: func(ctx context.Context, shortID, originalURL, userID string) error {
			return nil
		},
		FindByOriginalURLFunc: func(ctx context.Context, originalURL string) (string, error) {
			return "", nil
		},
	}

	fetcher := &MockURLFetcher{
		GetURLsByUserIDFunc: func(ctx context.Context, uid string) ([]models.UserURL, error) {
			return []models.UserURL{
				{
					ShortURL:    "existing",
					OriginalURL: "https://existing.com",
					UserID:      userID,
				},
			}, nil
		},
	}

	generator := &MockGenerator{
		GenerateFunc: func() string {
			return shortID
		},
	}

	service := NewService(
		saver,
		&MockURLBatchSaver{},
		&MockURLGetter{},
		fetcher,
		&MockURLDeleter{},
		&MockPinger{},
		generator,
		"http://short.url",
	)

	_, _ = service.GetURLsByUserID(ctx, userID)

	service.cacheMu.RLock()
	if len(service.cache[userID]) == 0 {
		t.Error("Cache should not be empty before ShortenURL")
	}
	service.cacheMu.RUnlock()

	_, err := service.ShortenURL(ctx, originalURL, userID)
	if err != nil {
		t.Fatalf("ShortenURL returned unexpected error: %v", err)
	}

	service.cacheMu.RLock()
	_, exists := service.cache[userID]
	service.cacheMu.RUnlock()

	if exists {
		t.Error("Cache should be cleared after ShortenURL")
	}
}

func TestService_CacheClearOnShortenBatch(t *testing.T) {
	ctx := context.Background()
	userID := "user1"

	batchSaver := &MockURLBatchSaver{
		SaveBatchFunc: func(ctx context.Context, items map[string]string, userID string) error {
			return nil
		},
	}

	fetcher := &MockURLFetcher{
		GetURLsByUserIDFunc: func(ctx context.Context, uid string) ([]models.UserURL, error) {
			return []models.UserURL{
				{
					ShortURL:    "existing",
					OriginalURL: "https://existing.com",
					UserID:      userID,
				},
			}, nil
		},
	}

	generator := &MockGenerator{
		GenerateFunc: func() string {
			return "generated"
		},
	}

	service := NewService(
		&MockURLSaver{},
		batchSaver,
		&MockURLGetter{},
		fetcher,
		&MockURLDeleter{},
		&MockPinger{},
		generator,
		"http://short.url",
	)

	_, _ = service.GetURLsByUserID(ctx, userID)

	service.cacheMu.RLock()
	if len(service.cache[userID]) == 0 {
		t.Error("Cache should not be empty before ShortenBatch")
	}
	service.cacheMu.RUnlock()

	batch := []models.BatchShortenRequest{
		{
			CorrelationID: "1",
			OriginalURL:   "https://example1.com",
		},
	}

	_, err := service.ShortenBatch(ctx, batch, userID)
	if err != nil {
		t.Fatalf("ShortenBatch returned unexpected error: %v", err)
	}

	service.cacheMu.RLock()
	_, exists := service.cache[userID]
	service.cacheMu.RUnlock()

	if exists {
		t.Error("Cache should be cleared after ShortenBatch")
	}
}
