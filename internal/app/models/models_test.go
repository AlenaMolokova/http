// Package models_test содержит тесты для структур и методов из пакета models.
package models_test

import (
	"encoding/json"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/stretchr/testify/assert"
)

// TestShortenResponse_MarshalJSON проверяет сериализацию ShortenResponse в JSON.
func TestShortenResponse_MarshalJSON(t *testing.T) {
	resp := models.ShortenResponse{Result: "http://short.url/abc"}
	data, err := json.Marshal(resp)

	assert.NoError(t, err)
	assert.JSONEq(t, `{"result":"http://short.url/abc"}`, string(data))
}

// TestShortenRequest_UnmarshalJSON проверяет десериализацию ShortenRequest из JSON.
func TestShortenRequest_UnmarshalJSON(t *testing.T) {
	data := []byte(`{"url":"https://example.com"}`)
	var req models.ShortenRequest

	err := json.Unmarshal(data, &req)

	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", req.URL)
}

// TestShortenRequest_UnmarshalJSON_Invalid проверяет обработку некорректного JSON.
func TestShortenRequest_UnmarshalJSON_Invalid(t *testing.T) {
	data := []byte(`{invalid_json`)
	var req models.ShortenRequest

	err := json.Unmarshal(data, &req)

	assert.Error(t, err)
}
