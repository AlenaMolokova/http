// Package middleware содержит middleware функции для HTTP-сервера.
package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// SubnetChecker представляет middleware для проверки доверенных подсетей.
type SubnetChecker struct {
	trustedSubnet *net.IPNet
}

// NewSubnetChecker создает новый экземпляр SubnetChecker.
// Если trustedSubnet пустая строка, middleware будет пропускать все запросы.
func NewSubnetChecker(trustedSubnet string) *SubnetChecker {
	if trustedSubnet == "" {
		return &SubnetChecker{trustedSubnet: nil}
	}

	_, subnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		logrus.WithError(err).WithField("subnet", trustedSubnet).Error("Invalid trusted subnet format")
		return &SubnetChecker{trustedSubnet: nil}
	}

	return &SubnetChecker{trustedSubnet: subnet}
}

// CheckSubnet возвращает middleware функцию для проверки IP-адреса клиента.
func (sc *SubnetChecker) CheckSubnet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если доверенная подсеть не настроена, пропускаем проверку
		if sc.trustedSubnet == nil {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := sc.getClientIP(r)
		if clientIP == nil {
			logrus.Warn("Unable to determine client IP address")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if !sc.trustedSubnet.Contains(clientIP) {
			logrus.WithFields(logrus.Fields{
				"client_ip":      clientIP.String(),
				"trusted_subnet": sc.trustedSubnet.String(),
			}).Warn("Access denied: IP not in trusted subnet")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP извлекает реальный IP-адрес клиента из HTTP-запроса.
// Учитывает заголовки X-Forwarded-For и X-Real-IP для работы с прокси.
func (sc *SubnetChecker) getClientIP(r *http.Request) net.IP {
	// Проверяем заголовок X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For может содержать несколько IP-адресов через запятую
		ips := strings.Split(xff, ",")
		for _, ipStr := range ips {
			ipStr = strings.TrimSpace(ipStr)
			if ip := net.ParseIP(ipStr); ip != nil {
				return ip
			}
		}
	}

	// Проверяем заголовок X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := net.ParseIP(strings.TrimSpace(xri)); ip != nil {
			return ip
		}
	}

	// Используем RemoteAddr как последний вариант
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Если не удалось разделить host:port, пробуем парсить как IP
		return net.ParseIP(r.RemoteAddr)
	}

	return net.ParseIP(host)
}
