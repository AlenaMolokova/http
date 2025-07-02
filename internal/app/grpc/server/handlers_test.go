// Package server содержит тесты для gRPC-сервера, обработчиков и перехватчиков сервиса сокращения URL.
package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockURLShortener реализует интерфейс URLShortener для тестирования.
type mockURLShortener struct {
	ShortenURLFunc func(ctx context.Context, originalURL, userID string) (models.ShortenResult, error)
}

// ShortenURL имитирует метод сокращения URL.
func (m *mockURLShortener) ShortenURL(ctx context.Context, originalURL, userID string) (models.ShortenResult, error) {
	return m.ShortenURLFunc(ctx, originalURL, userID)
}

// mockBatchURLShortener реализует интерфейс BatchURLShortener для тестирования.
type mockBatchURLShortener struct {
	ShortenBatchFunc func(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error)
}

// ShortenBatch имитирует метод пакетного сокращения URL.
func (m *mockBatchURLShortener) ShortenBatch(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
	return m.ShortenBatchFunc(ctx, items, userID)
}

// mockURLGetter реализует интерфейс URLGetter для тестирования.
type mockURLGetter struct {
	GetFunc func(ctx context.Context, shortID string) (string, bool)
}

// Get имитирует метод получения оригинального URL по короткому ID.
func (m *mockURLGetter) Get(ctx context.Context, shortID string) (string, bool) {
	return m.GetFunc(ctx, shortID)
}

// mockURLFetcher реализует интерфейс URLFetcher для тестирования.
type mockURLFetcher struct {
	GetURLsByUserIDFunc func(ctx context.Context, userID string) ([]models.UserURL, error)
}

// GetURLsByUserID имитирует метод получения всех URL пользователя.
func (m *mockURLFetcher) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	return m.GetURLsByUserIDFunc(ctx, userID)
}

// mockURLDeleter реализует интерфейс URLDeleter для тестирования.
type mockURLDeleter struct {
	DeleteURLsFunc func(ctx context.Context, shortIDs []string, userID string) error
}

// DeleteURLs имитирует метод удаления URL пользователя.
func (m *mockURLDeleter) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	return m.DeleteURLsFunc(ctx, shortIDs, userID)
}

// mockPinger реализует интерфейс Pinger для тестирования.
type mockPinger struct {
	PingFunc func(ctx context.Context) error
}

// Ping имитирует метод проверки соединения с хранилищем.
func (m *mockPinger) Ping(ctx context.Context) error {
	return m.PingFunc(ctx)
}

// mockStatsProvider реализует интерфейс StatsProvider для тестирования.
type mockStatsProvider struct {
	GetStatsFunc func(ctx context.Context) (models.Stats, error)
}

// GetStats имитирует метод получения статистики сервиса.
func (m *mockStatsProvider) GetStats(ctx context.Context) (models.Stats, error) {
	return m.GetStatsFunc(ctx)
}

// TestShortenerHandler_ShortenURL тестирует метод ShortenURL обработчика ShortenerHandler.
// Проверяет обработку корректных и некорректных запросов, а также ошибки зависимостей.
func TestShortenerHandler_ShortenURL(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.ShortenURLRequest
		setupMock      func(m *mockURLShortener)
		expectedResp   *pb.ShortenURLResponse
		expectedErr    error
		expectedStatus codes.Code
	}{
		{
			name: "Valid URL",
			req:  &pb.ShortenURLRequest{Url: "https://example.com", UserId: "user123"},
			setupMock: func(m *mockURLShortener) {
				m.ShortenURLFunc = func(ctx context.Context, originalURL, userID string) (models.ShortenResult, error) {
					return models.ShortenResult{ShortURL: "http://short.url/abc123", IsNew: true}, nil
				}
			},
			expectedResp:   &pb.ShortenURLResponse{ShortUrl: "http://short.url/abc123", IsNew: true},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name: "Empty URL",
			req:  &pb.ShortenURLRequest{Url: "", UserId: "user123"},
			setupMock: func(m *mockURLShortener) {
				m.ShortenURLFunc = func(ctx context.Context, originalURL, userID string) (models.ShortenResult, error) {
					return models.ShortenResult{}, nil
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "URL cannot be empty"),
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "Invalid URL",
			req:  &pb.ShortenURLRequest{Url: "invalid-url", UserId: "user123"},
			setupMock: func(m *mockURLShortener) {
				m.ShortenURLFunc = func(ctx context.Context, originalURL, userID string) (models.ShortenResult, error) {
					return models.ShortenResult{}, nil
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "Invalid URL format"),
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "Shortener Error",
			req:  &pb.ShortenURLRequest{Url: "https://example.com", UserId: "user123"},
			setupMock: func(m *mockURLShortener) {
				m.ShortenURLFunc = func(ctx context.Context, originalURL, userID string) (models.ShortenResult, error) {
					return models.ShortenResult{}, status.Error(codes.Internal, "shortener failed")
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.Internal, "Failed to shorten URL"),
			expectedStatus: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockShortener := &mockURLShortener{}
			tt.setupMock(mockShortener)

			handler := NewShortenerHandler(mockShortener, nil, nil, nil, nil, nil, nil, "http://short.url")
			resp, err := handler.ShortenURL(context.Background(), tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedStatus, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

// TestShortenerHandler_ShortenURLJSON тестирует метод ShortenURLJSON обработчика ShortenerHandler.
// Проверяет обработку корректных и некорректных JSON-запросов, а также ошибки зависимостей.
func TestShortenerHandler_ShortenURLJSON(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.ShortenURLJSONRequest
		setupMock      func(m *mockURLShortener)
		expectedResp   *pb.ShortenURLJSONResponse
		expectedErr    error
		expectedStatus codes.Code
	}{
		{
			name: "Valid URL",
			req:  &pb.ShortenURLJSONRequest{Url: "https://example.com", UserId: "user123"},
			setupMock: func(m *mockURLShortener) {
				m.ShortenURLFunc = func(ctx context.Context, originalURL, userID string) (models.ShortenResult, error) {
					return models.ShortenResult{ShortURL: "http://short.url/abc123", IsNew: true}, nil
				}
			},
			expectedResp:   &pb.ShortenURLJSONResponse{Result: "http://short.url/abc123", IsNew: true},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name:           "Empty URL",
			req:            &pb.ShortenURLJSONRequest{Url: "", UserId: "user123"},
			setupMock:      func(m *mockURLShortener) {}, // Не требуется мок
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "URL cannot be empty"),
			expectedStatus: codes.InvalidArgument,
		},
		{
			name:           "Invalid URL",
			req:            &pb.ShortenURLJSONRequest{Url: "invalid-url", UserId: "user123"},
			setupMock:      func(m *mockURLShortener) {}, // Не требуется мок
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "Invalid URL format"),
			expectedStatus: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockShortener := &mockURLShortener{}
			tt.setupMock(mockShortener)

			handler := NewShortenerHandler(mockShortener, nil, nil, nil, nil, nil, nil, "http://short.url")
			resp, err := handler.ShortenURLJSON(context.Background(), tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedStatus, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

// TestShortenerHandler_BatchShortenURL тестирует метод BatchShortenURL обработчика ShortenerHandler.
// Проверяет обработку корректных и некорректных пакетных запросов, а также ошибки зависимостей.
func TestShortenerHandler_BatchShortenURL(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.BatchShortenURLRequest
		setupMock      func(m *mockBatchURLShortener)
		expectedResp   *pb.BatchShortenURLResponse
		expectedErr    error
		expectedStatus codes.Code
	}{
		{
			name: "Valid Batch",
			req: &pb.BatchShortenURLRequest{
				Items:  []*pb.BatchItem{{CorrelationId: "1", OriginalUrl: "https://example.com"}},
				UserId: "user123",
			},
			setupMock: func(m *mockBatchURLShortener) {
				m.ShortenBatchFunc = func(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
					return []models.BatchShortenResponse{
						{CorrelationID: "1", ShortURL: "http://short.url/abc123"},
					}, nil
				}
			},
			expectedResp: &pb.BatchShortenURLResponse{
				Items: []*pb.BatchResponseItem{
					{CorrelationId: "1", ShortUrl: "http://short.url/abc123"},
				},
			},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name:           "Empty Batch",
			req:            &pb.BatchShortenURLRequest{Items: []*pb.BatchItem{}, UserId: "user123"},
			setupMock:      func(m *mockBatchURLShortener) {}, // не нужен мок
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "Empty batch"),
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "Invalid URL in Batch",
			req: &pb.BatchShortenURLRequest{
				Items:  []*pb.BatchItem{{CorrelationId: "1", OriginalUrl: "invalid-url"}},
				UserId: "user123",
			},
			setupMock: func(m *mockBatchURLShortener) {
				// Исправлено: убрали panic, потому что метод не должен вызываться при валидации
				m.ShortenBatchFunc = func(ctx context.Context, items []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
					return nil, fmt.Errorf("should not be called")
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "Invalid URL format"),
			expectedStatus: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBatch := &mockBatchURLShortener{}
			tt.setupMock(mockBatch)

			handler := NewShortenerHandler(nil, mockBatch, nil, nil, nil, nil, nil, "http://short.url")
			resp, err := handler.BatchShortenURL(context.Background(), tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedStatus, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

// TestShortenerHandler_GetURL тестирует метод GetURL обработчика ShortenerHandler.
// Проверяет получение оригинального URL по короткому ID, включая случаи, когда URL не найден.
func TestShortenerHandler_GetURL(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.GetURLRequest
		setupMock      func(m *mockURLGetter)
		expectedResp   *pb.GetURLResponse
		expectedErr    error
		expectedStatus codes.Code
	}{
		{
			name: "URL Found",
			req:  &pb.GetURLRequest{Id: "abc123"},
			setupMock: func(m *mockURLGetter) {
				m.GetFunc = func(ctx context.Context, shortID string) (string, bool) {
					return "https://example.com", true
				}
			},
			expectedResp:   &pb.GetURLResponse{OriginalUrl: "https://example.com", Found: true},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name: "URL Not Found",
			req:  &pb.GetURLRequest{Id: "abc123"},
			setupMock: func(m *mockURLGetter) {
				m.GetFunc = func(ctx context.Context, shortID string) (string, bool) {
					return "", false
				}
			},
			expectedResp:   &pb.GetURLResponse{OriginalUrl: "", Found: false},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name: "Empty ID",
			req:  &pb.GetURLRequest{Id: ""},
			setupMock: func(m *mockURLGetter) {
				m.GetFunc = func(ctx context.Context, shortID string) (string, bool) {
					return "", false
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "ID cannot be empty"),
			expectedStatus: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGetter := &mockURLGetter{}
			tt.setupMock(mockGetter)

			handler := NewShortenerHandler(nil, nil, mockGetter, nil, nil, nil, nil, "")
			resp, err := handler.GetURL(context.Background(), tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedStatus, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

// TestShortenerHandler_GetUserURLs тестирует метод GetUserURLs обработчика ShortenerHandler.
// Проверяет получение всех URL пользователя, включая случаи с некорректным ID и ошибками зависимостей.
func TestShortenerHandler_GetUserURLs(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.GetUserURLsRequest
		setupMock      func(m *mockURLFetcher)
		expectedResp   *pb.GetUserURLsResponse
		expectedErr    error
		expectedStatus codes.Code
	}{
		{
			name: "Valid User",
			req:  &pb.GetUserURLsRequest{UserId: "user123"},
			setupMock: func(m *mockURLFetcher) {
				m.GetURLsByUserIDFunc = func(ctx context.Context, userID string) ([]models.UserURL, error) {
					return []models.UserURL{
						{ShortURL: "http://short.url/abc123", OriginalURL: "https://example.com"},
					}, nil
				}
			},
			expectedResp: &pb.GetUserURLsResponse{
				Urls: []*pb.UserURL{
					{ShortUrl: "http://short.url/abc123", OriginalUrl: "https://example.com"},
				},
			},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name: "Empty User ID",
			req:  &pb.GetUserURLsRequest{UserId: ""},
			setupMock: func(m *mockURLFetcher) {
				m.GetURLsByUserIDFunc = func(ctx context.Context, userID string) ([]models.UserURL, error) {
					return nil, nil
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "User ID cannot be empty"),
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "Fetcher Error",
			req:  &pb.GetUserURLsRequest{UserId: "user123"},
			setupMock: func(m *mockURLFetcher) {
				m.GetURLsByUserIDFunc = func(ctx context.Context, userID string) ([]models.UserURL, error) {
					return nil, fmt.Errorf("fetcher error")
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.Internal, "Failed to get user URLs"),
			expectedStatus: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFetcher := &mockURLFetcher{}
			tt.setupMock(mockFetcher)

			handler := NewShortenerHandler(nil, nil, nil, mockFetcher, nil, nil, nil, "")
			resp, err := handler.GetUserURLs(context.Background(), tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedStatus, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

// TestShortenerHandler_DeleteURLs тестирует метод DeleteURLs обработчика ShortenerHandler.
// Проверяет удаление URL пользователя, включая случаи с некорректными данными и ошибками зависимостей.
func TestShortenerHandler_DeleteURLs(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.DeleteURLsRequest
		setupMock      func(m *mockURLDeleter)
		expectedResp   *pb.DeleteURLsResponse
		expectedErr    error
		expectedStatus codes.Code
	}{
		{
			name: "Valid Deletion",
			req:  &pb.DeleteURLsRequest{ShortIds: []string{"abc123"}, UserId: "user123"},
			setupMock: func(m *mockURLDeleter) {
				m.DeleteURLsFunc = func(ctx context.Context, shortIDs []string, userID string) error {
					return nil
				}
			},
			expectedResp:   &pb.DeleteURLsResponse{Success: true, Message: "URLs deleted successfully"},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name: "Empty Short IDs",
			req:  &pb.DeleteURLsRequest{ShortIds: []string{}, UserId: "user123"},
			setupMock: func(m *mockURLDeleter) {
				m.DeleteURLsFunc = func(ctx context.Context, shortIDs []string, userID string) error {
					return nil
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "Empty list of URLs"),
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "Empty User ID",
			req:  &pb.DeleteURLsRequest{ShortIds: []string{"abc123"}, UserId: ""},
			setupMock: func(m *mockURLDeleter) {
				m.DeleteURLsFunc = func(ctx context.Context, shortIDs []string, userID string) error {
					return nil
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.InvalidArgument, "User ID cannot be empty"),
			expectedStatus: codes.InvalidArgument,
		},
		{
			name: "Deleter Error",
			req:  &pb.DeleteURLsRequest{ShortIds: []string{"abc123"}, UserId: "user123"},
			setupMock: func(m *mockURLDeleter) {
				m.DeleteURLsFunc = func(ctx context.Context, shortIDs []string, userID string) error {
					return fmt.Errorf("deleter error")
				}
			},
			expectedResp:   &pb.DeleteURLsResponse{Success: false, Message: "Failed to delete URLs"},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDeleter := &mockURLDeleter{}
			tt.setupMock(mockDeleter)

			handler := NewShortenerHandler(nil, nil, nil, nil, mockDeleter, nil, nil, "")
			resp, err := handler.DeleteURLs(context.Background(), tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedStatus, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}

// TestShortenerHandler_Ping тестирует метод Ping обработчика ShortenerHandler.
// Проверяет проверку соединения с хранилищем, включая случаи успешного и неуспешного пинга.
func TestShortenerHandler_Ping(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.PingRequest
		setupMock      func(m *mockPinger)
		expectedResp   *pb.PingResponse
		expectedErr    error
		expectedStatus codes.Code
	}{
		{
			name: "Ping Success",
			req:  &pb.PingRequest{},
			setupMock: func(m *mockPinger) {
				m.PingFunc = func(ctx context.Context) error {
					return nil
				}
			},
			expectedResp:   &pb.PingResponse{Status: "OK", Message: "Database connection is OK"},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name: "Ping No DB Support",
			req:  &pb.PingRequest{},
			setupMock: func(m *mockPinger) {
				m.PingFunc = func(ctx context.Context) error {
					return fmt.Errorf("does not support database connection check")
				}
			},
			expectedResp:   &pb.PingResponse{Status: "OK", Message: "Storage does not require database connection"},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name: "Ping Failure",
			req:  &pb.PingRequest{},
			setupMock: func(m *mockPinger) {
				m.PingFunc = func(ctx context.Context) error {
					return fmt.Errorf("connection error")
				}
			},
			expectedResp:   &pb.PingResponse{Status: "ERROR", Message: "Database connection error"},
			expectedErr:    status.Error(codes.Internal, "Database connection error"),
			expectedStatus: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPinger := &mockPinger{}
			tt.setupMock(mockPinger)

			handler := NewShortenerHandler(nil, nil, nil, nil, nil, mockPinger, nil, "")
			resp, err := handler.Ping(context.Background(), tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedStatus, status.Code(err))
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResp, resp)
		})
	}
}

// TestShortenerHandler_GetStats тестирует метод GetStats обработчика ShortenerHandler.
// Проверяет получение статистики сервиса, включая случаи успеха и ошибки зависимости.
func TestShortenerHandler_GetStats(t *testing.T) {
	tests := []struct {
		name           string
		req            *pb.GetStatsRequest
		setupMock      func(m *mockStatsProvider)
		expectedResp   *pb.GetStatsResponse
		expectedErr    error
		expectedStatus codes.Code
	}{
		{
			name: "Valid Stats",
			req:  &pb.GetStatsRequest{},
			setupMock: func(m *mockStatsProvider) {
				m.GetStatsFunc = func(ctx context.Context) (models.Stats, error) {
					return models.Stats{URLs: 100, Users: 50}, nil
				}
			},
			expectedResp:   &pb.GetStatsResponse{Urls: 100, Users: 50},
			expectedErr:    nil,
			expectedStatus: codes.OK,
		},
		{
			name: "Stats Error",
			req:  &pb.GetStatsRequest{},
			setupMock: func(m *mockStatsProvider) {
				m.GetStatsFunc = func(ctx context.Context) (models.Stats, error) {
					return models.Stats{}, fmt.Errorf("stats error")
				}
			},
			expectedResp:   nil,
			expectedErr:    status.Error(codes.Internal, "Failed to get stats"),
			expectedStatus: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStats := &mockStatsProvider{}
			tt.setupMock(mockStats)

			handler := NewShortenerHandler(nil, nil, nil, nil, nil, nil, mockStats, "")
			resp, err := handler.GetStats(context.Background(), tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedStatus, status.Code(err))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}
		})
	}
}
