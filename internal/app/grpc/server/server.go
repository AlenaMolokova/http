// Package server предоставляет реализацию gRPC-сервера для сервиса сокращения URL.
// Содержит сервер, обработчики и интерцепторы для обработки gRPC-запросов.
package server

import (
	"net"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCServer представляет gRPC-сервер для сервиса сокращения URL.
// Хранит сервер, слушатель сети и обработчик запросов.
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	handler  *ShortenerHandler
}

// ShortenerHandler содержит бизнес-логику для gRPC-методов.
// Реализует интерфейс ShortenerServiceServer из сгенерированного protobuf.
type ShortenerHandler struct {
	pb.UnimplementedShortenerServiceServer

	shortener     models.URLShortener
	batch         models.BatchURLShortener
	getter        models.URLGetter
	fetcher       models.URLFetcher
	deleter       models.URLDeleter
	pinger        models.Pinger
	statsProvider models.StatsProvider
	baseURL       string
}

// NewShortenerHandler создает новый обработчик gRPC-запросов.
//
// Параметры:
//   - shortener: интерфейс для сокращения URL
//   - batch: интерфейс для пакетного сокращения URL
//   - getter: интерфейс для получения оригинального URL
//   - fetcher: интерфейс для получения URL пользователя
//   - deleter: интерфейс для удаления URL
//   - pinger: интерфейс для проверки соединения с хранилищем
//   - statsProvider: интерфейс для получения статистики
//   - baseURL: базовый URL для сокращенных ссылок
//
// Возвращает: указатель на ShortenerHandler.
func NewShortenerHandler(
	shortener models.URLShortener,
	batch models.BatchURLShortener,
	getter models.URLGetter,
	fetcher models.URLFetcher,
	deleter models.URLDeleter,
	pinger models.Pinger,
	statsProvider models.StatsProvider,
	baseURL string,
) *ShortenerHandler {
	return &ShortenerHandler{
		shortener:     shortener,
		batch:         batch,
		getter:        getter,
		fetcher:       fetcher,
		deleter:       deleter,
		pinger:        pinger,
		statsProvider: statsProvider,
		baseURL:       baseURL,
	}
}

// NewGRPCServer создает новый gRPC-сервер с заданной конфигурацией.
//
// Параметры:
//   - address: адрес для прослушивания (например, ":3201")
//   - handler: обработчик gRPC-запросов
//   - cfg: конфигурация приложения
//   - logger: логгер для записи событий
//   - enableTLS: флаг включения TLS
//   - certFile: путь к файлу TLS-сертификата
//   - keyFile: путь к файлу TLS-приватного ключа
//
// Возвращает:
//   - указатель на GRPCServer
//   - ошибку, если сервер не удалось создать
func NewGRPCServer(
	address string,
	handler *ShortenerHandler,
	cfg *config.Config,
	logger *logrus.Logger,
	enableTLS bool,
	certFile, keyFile string,
) (*GRPCServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	var opts []grpc.ServerOption

	// Настройка TLS, если включена
	if enableTLS && certFile != "" && keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Настройка интерцепторов
	var trustedSubnet *net.IPNet
	if cfg.TrustedSubnet != "" {
		_, ipNet, err := net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			logger.Errorf("Failed to parse trusted subnet: %v", err)
			return nil, err
		}
		trustedSubnet = ipNet
	}
	opts = append(opts, grpc.ChainUnaryInterceptor(
		LoggingInterceptor(logger),
		SubnetInterceptor(trustedSubnet),
	))

	// Создаем сервер с интерцепторами
	server := grpc.NewServer(opts...)

	// Регистрируем сервис
	pb.RegisterShortenerServiceServer(server, handler)

	return &GRPCServer{
		server:   server,
		listener: listener,
		handler:  handler,
	}, nil
}

// Start запускает gRPC-сервер.
//
// Возвращает: ошибку, если сервер не удалось запустить.
func (s *GRPCServer) Start() error {
	logrus.Infof("Starting gRPC server on %s", s.listener.Addr().String())
	return s.server.Serve(s.listener)
}

// Stop останавливает gRPC-сервер с graceful shutdown.
func (s *GRPCServer) Stop() {
	logrus.Info("Stopping gRPC server")
	s.server.GracefulStop()
}

// GetAddress возвращает адрес, на котором работает gRPC-сервер.
//
// Возвращает: строковый адрес сервера.
func (s *GRPCServer) GetAddress() string {
	return s.listener.Addr().String()
}
