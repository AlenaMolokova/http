package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	
	"github.com/google/uuid"
)

var SecretKey = []byte("your-secret-key-change-this-in-production")

const (
	CookieName = "user_id"
	CookieMaxAge = 30 * 24 * 60 * 60
)

func GenerateUserID() string {
	return uuid.New().String()
}

func SignData(data string) string {
	h := hmac.New(sha256.New, SecretKey)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func VerifySignature(data, signature string) bool {
	expectedSignature := SignData(data)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func GetUserIDFromCookie(r *http.Request) (string, error) {
	parts := make(map[string]string)
	for _, part := range []string{"id", "sign"} {
		cookie, err := r.Cookie(fmt.Sprintf("%s_%s", CookieName, part))
		if err != nil {
			return "", errors.New("invalid cookie format")
		}
		parts[part] = cookie.Value
	}

	userID := parts["id"]
	signature := parts["sign"]

	if !VerifySignature(userID, signature) {
		return "", errors.New("invalid signature")
	}

	return userID, nil
}

func SetUserIDCookie(w http.ResponseWriter, userID string) {
	signature := SignData(userID)

	http.SetCookie(w, &http.Cookie{
		Name:     fmt.Sprintf("%s_id", CookieName),
		Value:    userID,
		Path:     "/",
		MaxAge:   CookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     fmt.Sprintf("%s_sign", CookieName),
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

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := GetUserIDFromCookie(r)
		
		if err != nil {
			userID = GenerateUserID()
			SetUserIDCookie(w, userID)
		}
		
		next.ServeHTTP(w, r)
	})
}