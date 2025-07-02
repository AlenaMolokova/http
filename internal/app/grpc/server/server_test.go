// Package server содержит тесты для gRPC-сервера, обработчиков и перехватчиков сервиса сокращения URL.
package server

import (
	"testing"
	"time"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestNewGRPCServer тестирует создание экземпляра GRPCServer.
// Проверяет инициализацию сервера с различными конфигурациями, включая TLS и доверенную подсеть.
func TestNewGRPCServer(t *testing.T) {
	logger := logrus.New()
	cfg := &config.Config{TrustedSubnet: "192.168.1.0/24"}

	t.Run("Without TLS", func(t *testing.T) {
		handler := NewShortenerHandler(nil, nil, nil, nil, nil, nil, nil, "")
		server, err := NewGRPCServer(":0", handler, cfg, logger, false, "", "")
		assert.NoError(t, err)
		assert.NotNil(t, server)
		assert.NotNil(t, server.listener)
		assert.NotNil(t, server.server)
		assert.Equal(t, handler, server.handler)
		server.Stop()
	})

	t.Run("With Invalid TLS", func(t *testing.T) {
		handler := NewShortenerHandler(nil, nil, nil, nil, nil, nil, nil, "")
		server, err := NewGRPCServer(":0", handler, cfg, logger, true, "invalid-cert.pem", "invalid-key.pem")
		assert.Error(t, err)
		assert.Nil(t, server)
	})

	t.Run("Invalid Address", func(t *testing.T) {
		handler := NewShortenerHandler(nil, nil, nil, nil, nil, nil, nil, "")
		server, err := NewGRPCServer("invalid-address", handler, cfg, logger, false, "", "")
		assert.Error(t, err)
		assert.Nil(t, server)
	})

	t.Run("Invalid Trusted Subnet", func(t *testing.T) {
		invalidCfg := &config.Config{TrustedSubnet: "invalid-subnet"}
		handler := NewShortenerHandler(nil, nil, nil, nil, nil, nil, nil, "")
		server, err := NewGRPCServer(":0", handler, invalidCfg, logger, false, "", "")
		assert.Error(t, err)
		assert.Nil(t, server)
	})
}

// TestGRPCServer_Lifecycle тестирует жизненный цикл GRPCServer.
// Проверяет запуск, получение адреса и остановку сервера.
func TestGRPCServer_Lifecycle(t *testing.T) {
	logger := logrus.New()
	cfg := &config.Config{TrustedSubnet: ""}

	handler := NewShortenerHandler(nil, nil, nil, nil, nil, nil, nil, "")
	server, err := NewGRPCServer(":0", handler, cfg, logger, false, "", "")
	assert.NoError(t, err)

	go func() {
		err := server.Start()
		assert.NoError(t, err)
	}()

	time.Sleep(100 * time.Millisecond) // Ожидание запуска сервера

	addr := server.GetAddress()
	assert.NotEmpty(t, addr)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err)
	defer conn.Close()

	server.Stop()
}
