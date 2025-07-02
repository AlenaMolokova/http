// Package server содержит тесты для gRPC-сервера, обработчиков и перехватчиков сервиса сокращения URL.
package server

import (
	"bytes"
	"context"
	"net"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TestLoggingInterceptor тестирует перехватчик LoggingInterceptor.
// Проверяет логирование успешных и неуспешных gRPC-запросов.
func TestLoggingInterceptor(t *testing.T) {
	logger := logrus.New()
	logEntries := &bytes.Buffer{}
	logger.SetOutput(logEntries)

	interceptor := LoggingInterceptor(logger)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	t.Run("Successful Request", func(t *testing.T) {
		logEntries.Reset()
		resp, err := interceptor(ctx, "request", info, handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.Contains(t, logEntries.String(), "Received gRPC request")
		assert.Contains(t, logEntries.String(), "gRPC request completed")
	})

	t.Run("Failed Request", func(t *testing.T) {
		logEntries.Reset()
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, status.Error(codes.Internal, "handler error")
		}
		resp, err := interceptor(ctx, "request", info, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Nil(t, resp)
		assert.Contains(t, logEntries.String(), "Received gRPC request")
		assert.Contains(t, logEntries.String(), "gRPC request failed")
	})
}

// TestSubnetInterceptor тестирует перехватчик SubnetInterceptor.
// Проверяет валидацию IP-адресов клиентов на основе доверенной подсети.
func TestSubnetInterceptor(t *testing.T) {
	_, trustedSubnet, _ := net.ParseCIDR("192.168.1.0/24")
	interceptor := SubnetInterceptor(trustedSubnet)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	t.Run("Valid IP", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.100"))
		resp, err := interceptor(ctx, "request", nil, handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})

	t.Run("Invalid IP", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "10.0.0.100"))
		resp, err := interceptor(ctx, "request", nil, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, err.Error(), "Request from untrusted subnet")
		assert.Nil(t, resp)
	})

	t.Run("No Metadata", func(t *testing.T) {
		resp, err := interceptor(context.Background(), "request", nil, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), "Missing metadata")
		assert.Nil(t, resp)
	})

	t.Run("No Client IP", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("other-key", "value"))
		resp, err := interceptor(ctx, "request", nil, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), "Missing client IP")
		assert.Nil(t, resp)
	})

	t.Run("Invalid Client IP", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "invalid-ip"))
		resp, err := interceptor(ctx, "request", nil, handler)
		assert.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), "Invalid client IP")
		assert.Nil(t, resp)
	})

	t.Run("No Trusted Subnet", func(t *testing.T) {
		interceptor := SubnetInterceptor(nil)
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.100"))
		resp, err := interceptor(ctx, "request", nil, handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
	})
}
