// Package main предоставляет утилиту для генерации самоподписанных сертификатов для разработки.
// Использование: go run cmd/gencert/main.go
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// generateCertificate создает самоподписанный сертификат для разработки.
// Сертификат будет действителен для localhost и локальных IP-адресов.
func generateCertificate() error {
	// Генерируем приватный ключ
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("ошибка генерации приватного ключа: %w", err)
	}

	// Создаем шаблон сертификата
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"URL Shortener Dev"},
			Country:       []string{"RU"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour), // Действителен 1 год
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:    []string{"localhost"},
	}

	// Генерируем сертификат
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("ошибка создания сертификата: %w", err)
	}

	// Сохраняем сертификат в файл
	certOut, err := os.Create("server.crt")
	if err != nil {
		return fmt.Errorf("ошибка создания файла сертификата: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("ошибка записи сертификата: %w", err)
	}

	// Сохраняем приватный ключ в файл
	keyOut, err := os.Create("server.key")
	if err != nil {
		return fmt.Errorf("ошибка создания файла ключа: %w", err)
	}
	defer keyOut.Close()

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("ошибка маршаллинга приватного ключа: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER}); err != nil {
		return fmt.Errorf("ошибка записи приватного ключа: %w", err)
	}

	fmt.Println("Сертификат и ключ успешно созданы:")
	fmt.Println("- server.crt (сертификат)")
	fmt.Println("- server.key (приватный ключ)")
	fmt.Println("\nИспользование:")
	fmt.Println("./shortener -s")
	fmt.Println("или")
	fmt.Println("ENABLE_HTTPS=true ./shortener")

	return nil
}

func main() {
	if err := generateCertificate(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
}
