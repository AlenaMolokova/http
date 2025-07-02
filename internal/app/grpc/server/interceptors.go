// Package server содержит реализацию gRPC-сервера, обработчиков и перехватчиков сервиса сокращения URL.
package server

import (
	"context"
	"net"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor создает gRPC-перехватчик для логирования запросов и ответов.
// Логирует начало и завершение (или ошибку) каждого gRPC-запроса с использованием предоставленного логгера.
func LoggingInterceptor(logger *logrus.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.WithFields(logrus.Fields{
			"method": info.FullMethod,
		}).Info("Received gRPC request")

		resp, err := handler(ctx, req)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"method": info.FullMethod,
				"error":  err,
			}).Error("gRPC request failed")
			return nil, err
		}

		logger.WithFields(logrus.Fields{
			"method": info.FullMethod,
		}).Info("gRPC request completed")
		return resp, nil
	}
}

// SubnetInterceptor создает gRPC-перехватчик для проверки IP-адреса клиента.
// Проверяет, находится ли IP-адрес клиента из метаданных запроса в доверенной подсети.
func SubnetInterceptor(trustedSubnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if trustedSubnet == nil {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "Missing metadata")
		}

		clientIPs := md.Get("x-real-ip")
		if len(clientIPs) == 0 {
			return nil, status.Error(codes.InvalidArgument, "Missing client IP")
		}

		clientIP := net.ParseIP(clientIPs[0])
		if clientIP == nil {
			return nil, status.Error(codes.InvalidArgument, "Invalid client IP")
		}

		if !trustedSubnet.Contains(clientIP) {
			return nil, status.Error(codes.PermissionDenied, "Request from untrusted subnet")
		}

		return handler(ctx, req)
	}
}
