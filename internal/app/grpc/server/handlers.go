// Package server содержит реализацию gRPC хендлеров.
package server

import (
	"context"
	"net/url"
	"strings"

	"github.com/AlenaMolokova/http/internal/app/auth"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ShortenURL обрабатывает запросы на сокращение URL (текстовый формат).
func (h *ShortenerHandler) ShortenURL(ctx context.Context, req *pb.ShortenURLRequest) (*pb.ShortenURLResponse, error) {
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "URL cannot be empty")
	}

	if _, err := url.ParseRequestURI(req.Url); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid URL format")
	}

	userID := req.UserId
	if userID == "" {
		userID = auth.GenerateUserID()
	}

	result, err := h.shortener.ShortenURL(ctx, req.Url, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to shorten URL")
	}

	return &pb.ShortenURLResponse{
		ShortUrl: result.ShortURL,
		IsNew:    result.IsNew,
	}, nil
}

// ShortenURLJSON обрабатывает запросы на сокращение URL (JSON формат).
func (h *ShortenerHandler) ShortenURLJSON(ctx context.Context, req *pb.ShortenURLJSONRequest) (*pb.ShortenURLJSONResponse, error) {
	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "URL cannot be empty")
	}

	if _, err := url.ParseRequestURI(req.Url); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid URL format")
	}

	userID := req.UserId
	if userID == "" {
		userID = auth.GenerateUserID()
	}

	result, err := h.shortener.ShortenURL(ctx, req.Url, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to shorten URL")
	}

	return &pb.ShortenURLJSONResponse{
		Result: result.ShortURL,
		IsNew:  result.IsNew,
	}, nil
}

// BatchShortenURL обрабатывает запросы на пакетное сокращение URL.
func (h *ShortenerHandler) BatchShortenURL(ctx context.Context, req *pb.BatchShortenURLRequest) (*pb.BatchShortenURLResponse, error) {
	if len(req.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Empty batch")
	}

	userID := req.UserId
	if userID == "" {
		userID = auth.GenerateUserID()
	}

	// Конвертируем gRPC запрос в модель для бизнес-логики
	batchReq := make([]models.BatchShortenRequest, len(req.Items))
	for i, item := range req.Items {
		if item.OriginalUrl == "" {
			return nil, status.Error(codes.InvalidArgument, "URL cannot be empty")
		}
		// Исправлено: используем url.ParseRequestURI вместо url.Parse для консистентности
		if _, err := url.ParseRequestURI(item.OriginalUrl); err != nil {
			return nil, status.Error(codes.InvalidArgument, "Invalid URL format")
		}

		batchReq[i] = models.BatchShortenRequest{
			CorrelationID: item.CorrelationId,
			OriginalURL:   item.OriginalUrl,
		}
	}

	resp, err := h.batch.ShortenBatch(ctx, batchReq, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to shorten batch")
	}

	// Конвертируем ответ модели в gRPC ответ
	items := make([]*pb.BatchResponseItem, len(resp))
	for i, item := range resp {
		items[i] = &pb.BatchResponseItem{
			CorrelationId: item.CorrelationID,
			ShortUrl:      item.ShortURL,
		}
	}

	return &pb.BatchShortenURLResponse{
		Items: items,
	}, nil
}

// GetURL получает оригинальный URL по короткому ID.
func (h *ShortenerHandler) GetURL(ctx context.Context, req *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "ID cannot be empty")
	}

	originalURL, found := h.getter.Get(ctx, req.Id)

	return &pb.GetURLResponse{
		OriginalUrl: originalURL,
		Found:       found,
	}, nil
}

// GetUserURLs получает все URL пользователя.
func (h *ShortenerHandler) GetUserURLs(ctx context.Context, req *pb.GetUserURLsRequest) (*pb.GetUserURLsResponse, error) {
	userID := req.UserId
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "User ID cannot be empty")
	}

	urls, err := h.fetcher.GetURLsByUserID(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to get user URLs")
	}

	grpcURLs := make([]*pb.UserURL, len(urls))
	for i, u := range urls {
		grpcURLs[i] = &pb.UserURL{
			ShortUrl:    u.ShortURL,
			OriginalUrl: u.OriginalURL,
		}
	}

	return &pb.GetUserURLsResponse{
		Urls: grpcURLs,
	}, nil
}

// DeleteURLs удаляет URL пользователя.
func (h *ShortenerHandler) DeleteURLs(ctx context.Context, req *pb.DeleteURLsRequest) (*pb.DeleteURLsResponse, error) {
	if len(req.ShortIds) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Empty list of URLs")
	}

	userID := req.UserId
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "User ID cannot be empty")
	}

	if err := h.deleter.DeleteURLs(ctx, req.ShortIds, userID); err != nil {
		return &pb.DeleteURLsResponse{
			Success: false,
			Message: "Failed to delete URLs",
		}, nil
	}

	return &pb.DeleteURLsResponse{
		Success: true,
		Message: "URLs deleted successfully",
	}, nil
}

// Ping проверяет соединение с хранилищем.
func (h *ShortenerHandler) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	err := h.pinger.Ping(ctx)
	if err != nil {
		// Проверяем специальные случаи
		if strings.Contains(err.Error(), "does not support database connection check") {
			return &pb.PingResponse{
				Status:  "OK",
				Message: "Storage does not require database connection",
			}, nil
		}
		return &pb.PingResponse{
			Status:  "ERROR",
			Message: "Database connection error",
		}, status.Error(codes.Internal, "Database connection error")
	}

	return &pb.PingResponse{
		Status:  "OK",
		Message: "Database connection is OK",
	}, nil
}

// GetStats получает статистику сервиса.
func (h *ShortenerHandler) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	stats, err := h.statsProvider.GetStats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to get stats")
	}

	return &pb.GetStatsResponse{
		Urls:  int64(stats.URLs),
		Users: int64(stats.Users),
	}, nil
}
