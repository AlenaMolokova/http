package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateUserID тестирует генерацию уникального идентификатора пользователя
func TestGenerateUserID(t *testing.T) {
	userID1 := GenerateUserID()
	userID2 := GenerateUserID()

	assert.NotEmpty(t, userID1)
	assert.NotEmpty(t, userID2)
	assert.NotEqual(t, userID1, userID2)

	// Проверка, что это валидный UUID
	_, err := uuid.Parse(userID1)
	assert.NoError(t, err)
}

// TestSignData тестирует создание HMAC-SHA256 подписи
func TestSignData(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"Простая строка", "test-user-id"},
		{"UUID", "550e8400-e29b-41d4-a716-446655440000"},
		{"Пустая строка", ""},
		{"Спецсимволы", "test@user#123$%^&*()"},
		{"Кириллица", "тест-пользователь-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signature := SignData(tt.data)

			assert.NotEmpty(t, signature)
			assert.Len(t, signature, 64) // SHA256 в hex = 64 символа

			// Проверка детерминированности
			signature2 := SignData(tt.data)
			assert.Equal(t, signature, signature2)
		})
	}
}

// TestVerifySignature тестирует проверку подписи данных
func TestVerifySignature(t *testing.T) {
	testData := "test-user-id"
	validSignature := SignData(testData)

	tests := []struct {
		name      string
		data      string
		signature string
		expected  bool
	}{
		{"Валидная подпись", testData, validSignature, true},
		{"Неверная подпись", testData, "invalid-signature", false},
		{"Неверные данные", "different-data", validSignature, false},
		{"Пустая подпись", testData, "", false},
		{"Пустые данные", "", SignData(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifySignature(tt.data, tt.signature)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSetUserIDCookie тестирует установку cookie с идентификатором пользователя
func TestSetUserIDCookie(t *testing.T) {
	userID := "test-user-123"
	w := httptest.NewRecorder()

	SetUserIDCookie(w, userID)

	resp := w.Result()
	defer resp.Body.Close()
	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 3) // user_id_id, user_id_sign, user_id

	cookieMap := make(map[string]*http.Cookie)
	for _, cookie := range cookies {
		cookieMap[cookie.Name] = cookie
	}

	// Проверка cookie с ID
	idCookie, exists := cookieMap[fmt.Sprintf("%s_%s", CookieName, CookiePartID)]
	require.True(t, exists)
	assert.Equal(t, userID, idCookie.Value)
	assert.Equal(t, "/", idCookie.Path)
	assert.Equal(t, CookieMaxAge, idCookie.MaxAge)
	assert.True(t, idCookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, idCookie.SameSite)

	// Проверка cookie с подписью
	signCookie, exists := cookieMap[fmt.Sprintf("%s_%s", CookieName, CookiePartSign)]
	require.True(t, exists)
	assert.Equal(t, SignData(userID), signCookie.Value)
	assert.Equal(t, "/", signCookie.Path)
	assert.Equal(t, CookieMaxAge, signCookie.MaxAge)
	assert.True(t, signCookie.HttpOnly)

	// Проверка основного cookie
	mainCookie, exists := cookieMap[CookieName]
	require.True(t, exists)
	assert.Equal(t, "1", mainCookie.Value)
}

// TestGetUserIDFromCookie тестирует извлечение идентификатора пользователя из cookie
func TestGetUserIDFromCookie(t *testing.T) {
	tests := []struct {
		name          string
		setupCookies  func(*http.Request)
		expectedError bool
		expectedID    string
	}{
		{
			name: "Валидные cookies",
			setupCookies: func(r *http.Request) {
				userID := "test-user-123"
				signature := SignData(userID)
				r.AddCookie(&http.Cookie{
					Name:  fmt.Sprintf("%s_%s", CookieName, CookiePartID),
					Value: userID,
				})
				r.AddCookie(&http.Cookie{
					Name:  fmt.Sprintf("%s_%s", CookieName, CookiePartSign),
					Value: signature,
				})
			},
			expectedError: false,
			expectedID:    "test-user-123",
		},
		{
			name: "Отсутствует cookie с ID",
			setupCookies: func(r *http.Request) {
				r.AddCookie(&http.Cookie{
					Name:  fmt.Sprintf("%s_%s", CookieName, CookiePartSign),
					Value: "some-signature",
				})
			},
			expectedError: true,
		},
		{
			name: "Отсутствует cookie с подписью",
			setupCookies: func(r *http.Request) {
				r.AddCookie(&http.Cookie{
					Name:  fmt.Sprintf("%s_%s", CookieName, CookiePartID),
					Value: "test-user-123",
				})
			},
			expectedError: true,
		},
		{
			name: "Неверная подпись",
			setupCookies: func(r *http.Request) {
				r.AddCookie(&http.Cookie{
					Name:  fmt.Sprintf("%s_%s", CookieName, CookiePartID),
					Value: "test-user-123",
				})
				r.AddCookie(&http.Cookie{
					Name:  fmt.Sprintf("%s_%s", CookieName, CookiePartSign),
					Value: "invalid-signature",
				})
			},
			expectedError: true,
		},
		{
			name:          "Отсутствуют все cookies",
			setupCookies:  func(r *http.Request) {},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			tt.setupCookies(req)

			userID, err := GetUserIDFromCookie(req)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, userID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, userID)
			}
		})
	}
}

// TestRequireAuth тестирует middleware обязательной авторизации
func TestRequireAuth(t *testing.T) {
	handlerCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	tests := []struct {
		name           string
		setupCookies   func(*http.Request)
		expectedStatus int
		handlerCalled  bool
	}{
		{
			name: "Авторизованный пользователь",
			setupCookies: func(r *http.Request) {
				userID := "test-user-123"
				signature := SignData(userID)
				r.AddCookie(&http.Cookie{
					Name:  fmt.Sprintf("%s_%s", CookieName, CookiePartID),
					Value: userID,
				})
				r.AddCookie(&http.Cookie{
					Name:  fmt.Sprintf("%s_%s", CookieName, CookiePartSign),
					Value: signature,
				})
			},
			expectedStatus: http.StatusOK,
			handlerCalled:  true,
		},
		{
			name:           "Неавторизованный пользователь",
			setupCookies:   func(r *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
			handlerCalled:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false
			req := httptest.NewRequest("GET", "/", nil)
			tt.setupCookies(req)

			w := httptest.NewRecorder()
			authHandler := RequireAuth(nextHandler)
			authHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.handlerCalled, handlerCalled)
		})
	}
}
