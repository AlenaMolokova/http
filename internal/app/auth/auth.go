package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// CookiePartKey представляет тип ключа для частей cookie аутентификации.
type CookiePartKey string

// ContextKey представляет тип ключа для значений в контексте запроса.
type ContextKey string

const (
	// CookiePartID ключ для части cookie, содержащей идентификатор пользователя.
	CookiePartID CookiePartKey = "id"
	// CookiePartSign ключ для части cookie, содержащей подпись.
	CookiePartSign CookiePartKey = "sign"
)

const (
	// UserIDKey ключ для хранения идентификатора пользователя в контексте запроса.
	UserIDKey ContextKey = "userID"
)

// SecretKey секретный ключ для генерации подписи cookie.
// В продакшн-окружении следует заменить на более надежный ключ.
var SecretKey = []byte("your-secret-key-change-this-in-production")

const (
	// CookieName базовое имя cookie для хранения информации о пользователе.
	CookieName = "user_id"
	// CookieMaxAge максимальное время жизни cookie в секундах (30 дней).
	CookieMaxAge = 30 * 24 * 60 * 60
)

// GenerateUserID создает новый уникальный идентификатор пользователя.
//
// Возвращает:
//   - string: новый уникальный идентификатор в формате UUID.
func GenerateUserID() string {
	return uuid.New().String()
}

// SignData создает HMAC-SHA256 подпись для заданных данных, используя секретный ключ.
//
// Параметры:
//   - data: строка данных для подписи
//
// Возвращает:
//   - string: HMAC-SHA256 подпись в шестнадцатеричном формате
func SignData(data string) string {
	h := hmac.New(sha256.New, SecretKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature проверяет, соответствует ли подпись заданным данным.
//
// Параметры:
//   - data: исходные данные
//   - signature: подпись для проверки
//
// Возвращает:
//   - bool: true, если подпись верна; false в противном случае
func VerifySignature(data, signature string) bool {
	expectedSignature := SignData(data)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// GetUserIDFromCookie извлекает и проверяет идентификатор пользователя из cookie запроса.
//
// Параметры:
//   - r: HTTP-запрос, содержащий cookie
//
// Возвращает:
//   - string: идентификатор пользователя, если он действителен
//   - error: ошибка, если cookie отсутствует или подпись недействительна
func GetUserIDFromCookie(r *http.Request) (string, error) {
	parts := make(map[CookiePartKey]string)
	for _, part := range []CookiePartKey{CookiePartID, CookiePartSign} {
		cookie, err := r.Cookie(fmt.Sprintf("%s_%s", CookieName, part))
		if err != nil {
			return "", errors.New("invalid cookie format")
		}
		parts[part] = cookie.Value
	}

	userID := parts[CookiePartID]
	signature := parts[CookiePartSign]

	if !VerifySignature(userID, signature) {
		return "", errors.New("invalid signature")
	}

	return userID, nil
}

// SetUserIDCookie устанавливает cookie с идентификатором пользователя и подписью.
//
// Параметры:
//   - w: HTTP-ответ для установки cookie
//   - userID: идентификатор пользователя для сохранения
func SetUserIDCookie(w http.ResponseWriter, userID string) {
	signature := SignData(userID)

	http.SetCookie(w, &http.Cookie{
		Name:     fmt.Sprintf("%s_%s", CookieName, CookiePartID),
		Value:    userID,
		Path:     "/",
		MaxAge:   CookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     fmt.Sprintf("%s_%s", CookieName, CookiePartSign),
		Value:    signature,
		Path:     "/",
		MaxAge:   CookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "1",
		Path:     "/",
		MaxAge:   CookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// RequireAuth middleware, требующий обязательной авторизации для доступа к обработчику.
// Если пользователь не авторизован, возвращается ошибка 401 Unauthorized.
//
// Параметры:
//   - next: следующий обработчик HTTP
//
// Возвращает:
//   - http.HandlerFunc: middleware функция
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := GetUserIDFromCookie(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// AuthMiddleware middleware для аутентификации пользователей.
// Проверяет наличие cookie с идентификатором пользователя.
// Если cookie отсутствует или недействителен, создает нового пользователя.
// Добавляет идентификатор пользователя в контекст запроса.
//
// Параметры:
//   - next: следующий обработчик HTTP
//
// Возвращает:
//   - http.Handler: middleware обработчик
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromCookie(r)
		if err != nil {
			userID = GenerateUserID()
			SetUserIDCookie(w, userID)
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
