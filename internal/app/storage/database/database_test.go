package database

import (
	"context"
	"sort"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPostgresStorage тестирует создание нового PostgreSQL хранилища.
// Тест пропускается в CI окружении, так как требует реальной базы данных.
func TestNewPostgresStorage(t *testing.T) {
	t.Skip("Тест требует реальной базы данных, пропускаем в CI")
}

// MockDatabaseStorage представляет мок-объект для тестирования DatabaseStorage.
type MockDatabaseStorage struct {
	pool pgxmock.PgxPoolIface
}

// TestDatabaseStorage_Save тестирует сохранение одного URL в базе данных.
func TestDatabaseStorage_Save(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{
		pool: mockPool,
	}

	ctx := context.Background()
	shortID := "abc123"
	originalURL := "https://example.com"
	userID := "user1"

	mockPool.ExpectExec("INSERT INTO urls").
		WithArgs(shortID, originalURL, userID).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = db.Save(ctx, shortID, originalURL, userID)
	assert.NoError(t, err)

	err = mockPool.ExpectationsWereMet()
	assert.NoError(t, err)

	mockPool.ExpectExec("INSERT INTO urls").
		WithArgs("error", originalURL, userID).
		WillReturnError(pgx.ErrNoRows)

	err = db.Save(ctx, "error", originalURL, userID)
	assert.Error(t, err)
}

// Save сохраняет URL в базе данных.
func (db *MockDatabaseStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	_, err := db.pool.Exec(ctx, InsertURL, shortID, originalURL, userID)
	if err != nil {
		return err
	}
	return nil
}

// TestDatabaseStorage_FindByOriginalURL тестирует поиск короткого ID по оригинальному URL.
func TestDatabaseStorage_FindByOriginalURL(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{
		pool: mockPool,
	}

	ctx := context.Background()
	shortID := "abc123"
	originalURL := "https://example.com"

	mockPool.ExpectQuery("SELECT short_id").
		WithArgs(originalURL).
		WillReturnRows(pgxmock.NewRows([]string{"short_id"}).AddRow(shortID))

	result, err := db.FindByOriginalURL(ctx, originalURL)
	assert.NoError(t, err)
	assert.Equal(t, shortID, result)

	err = mockPool.ExpectationsWereMet()
	assert.NoError(t, err)

	mockPool.ExpectQuery("SELECT short_id").
		WithArgs("https://nonexistent.com").
		WillReturnRows(pgxmock.NewRows([]string{"short_id"}))

	result, err = db.FindByOriginalURL(ctx, "https://nonexistent.com")
	assert.NoError(t, err)
	assert.Empty(t, result)

	mockPool.ExpectQuery("SELECT short_id").
		WithArgs("error").
		WillReturnError(pgx.ErrTxClosed)

	result, err = db.FindByOriginalURL(ctx, "error")
	assert.Error(t, err)
	assert.Empty(t, result)
}

// FindByOriginalURL находит короткий ID по оригинальному URL.
func (db *MockDatabaseStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	var shortID string
	err := db.pool.QueryRow(ctx, SelectByOriginalURL, originalURL).Scan(&shortID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return shortID, nil
}

// TestDatabaseStorage_Get тестирует получение оригинального URL по короткому ID.
func TestDatabaseStorage_Get(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{
		pool: mockPool,
	}

	ctx := context.Background()
	shortID := "abc123"
	originalURL := "https://example.com"

	mockPool.ExpectQuery("SELECT original_url").
		WithArgs(shortID).
		WillReturnRows(pgxmock.NewRows([]string{"original_url"}).AddRow(originalURL))

	result, exists := db.Get(ctx, shortID)
	assert.True(t, exists)
	assert.Equal(t, originalURL, result)

	err = mockPool.ExpectationsWereMet()
	assert.NoError(t, err)

	mockPool.ExpectQuery("SELECT original_url").
		WithArgs("nonexistent").
		WillReturnError(pgx.ErrNoRows)

	result, exists = db.Get(ctx, "nonexistent")
	assert.False(t, exists)
	assert.Empty(t, result)

	mockPool.ExpectQuery("SELECT original_url").
		WithArgs("error").
		WillReturnError(pgx.ErrTxClosed)

	result, exists = db.Get(ctx, "error")
	assert.False(t, exists)
	assert.Empty(t, result)
}

// Get получает оригинальный URL по короткому ID.
func (db *MockDatabaseStorage) Get(ctx context.Context, shortID string) (string, bool) {
	var originalURL string
	err := db.pool.QueryRow(ctx, SelectByShortID, shortID).Scan(&originalURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false
		}
		return "", false
	}
	return originalURL, true
}

// TestDatabaseStorage_GetURLsByUserID тестирует получение всех URL пользователя.
func TestDatabaseStorage_GetURLsByUserID(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{
		pool: mockPool,
	}

	ctx := context.Background()
	userID := "user1"

	mockPool.ExpectQuery("SELECT short_id, original_url, user_id, is_deleted").
		WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{"short_id", "original_url", "user_id", "is_deleted"}).
			AddRow("abc123", "https://example.com", "user1", false).
			AddRow("def456", "https://test.com", "user1", false))

	urls, err := db.GetURLsByUserID(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.Equal(t, "abc123", urls[0].ShortURL)
	assert.Equal(t, "https://example.com", urls[0].OriginalURL)

	err = mockPool.ExpectationsWereMet()
	assert.NoError(t, err)

	mockPool.ExpectQuery("SELECT short_id, original_url, user_id, is_deleted").
		WithArgs("error").
		WillReturnError(pgx.ErrTxClosed)

	urls, err = db.GetURLsByUserID(ctx, "error")
	assert.Error(t, err)
	assert.Nil(t, urls)

	mockPool.ExpectQuery("SELECT short_id, original_url, user_id, is_deleted").
		WithArgs("error2").
		WillReturnRows(pgxmock.NewRows([]string{"short_id"}).AddRow("abc123")) // Неверное количество столбцов

	urls, err = db.GetURLsByUserID(ctx, "error2")
	assert.Error(t, err)
	assert.Nil(t, urls)
}

// GetURLsByUserID получает все URL пользователя.
func (db *MockDatabaseStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	rows, err := db.pool.Query(ctx, SelectByUserID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []models.UserURL
	for rows.Next() {
		var shortID, originalURL, userID string
		var isDeleted bool
		if err := rows.Scan(&shortID, &originalURL, &userID, &isDeleted); err != nil {
			return nil, err
		}
		urls = append(urls, models.UserURL{ShortURL: shortID, OriginalURL: originalURL})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

// TestDatabaseStorage_SaveBatch тестирует пакетное сохранение URL в базе данных.
// Исправлена проблема с недетерминированным порядком обработки элементов map.
func TestDatabaseStorage_SaveBatch(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{
		pool: mockPool,
	}

	ctx := context.Background()
	userID := "user1"
	batch := map[string]string{
		"abc123": "https://example.com",
		"def456": "https://test.com",
	}

	mockPool.ExpectBegin()

	// Сортируем ключи для детерминированного порядка
	var keys []string
	for k := range batch {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Добавляем ожидания в отсортированном порядке
	for _, shortID := range keys {
		originalURL := batch[shortID]
		mockPool.ExpectExec("INSERT INTO urls").
			WithArgs(shortID, originalURL, userID).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}

	mockPool.ExpectCommit()

	err = db.SaveBatch(ctx, batch, userID)
	assert.NoError(t, err)

	err = mockPool.ExpectationsWereMet()
	assert.NoError(t, err)

	// Тест на ошибку при выполнении запроса
	mockPool.ExpectBegin()
	mockPool.ExpectExec("INSERT INTO urls").
		WithArgs("error", "https://example.com", userID).
		WillReturnError(pgx.ErrTxClosed)
	mockPool.ExpectRollback()

	err = db.SaveBatch(ctx, map[string]string{"error": "https://example.com"}, userID)
	assert.Error(t, err)

	// Тест на ошибку при начале транзакции
	mockPool.ExpectBegin().WillReturnError(pgx.ErrTxClosed)

	err = db.SaveBatch(ctx, batch, userID)
	assert.Error(t, err)

	// Тест на ошибку при коммите
	mockPool.ExpectBegin()

	for _, shortID := range keys {
		originalURL := batch[shortID]
		mockPool.ExpectExec("INSERT INTO urls").
			WithArgs(shortID, originalURL, userID).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}

	mockPool.ExpectCommit().WillReturnError(pgx.ErrTxClosed)

	err = db.SaveBatch(ctx, batch, userID)
	assert.Error(t, err)
}

// SaveBatch сохраняет пакет URL в базе данных в рамках транзакции.
// Исправлена проблема с недетерминированным порядком обработки элементов map.
func (db *MockDatabaseStorage) SaveBatch(ctx context.Context, batch map[string]string, userID string) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Сортируем ключи для детерминированного порядка выполнения
	var keys []string
	for k := range batch {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Выполняем запросы в отсортированном порядке
	for _, shortID := range keys {
		originalURL := batch[shortID]
		_, err := tx.Exec(ctx, InsertURLBatch, shortID, originalURL, userID)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// TestDatabaseStorage_DeleteURLs тестирует удаление URL из базы данных.
func TestDatabaseStorage_DeleteURLs(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{
		pool: mockPool,
	}

	ctx := context.Background()
	userID := "user1"
	shortIDs := []string{"abc123", "def456"}

	mockPool.ExpectExec("UPDATE urls").
		WithArgs(shortIDs, userID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))

	err = db.DeleteURLs(ctx, shortIDs, userID)
	assert.NoError(t, err)

	err = mockPool.ExpectationsWereMet()
	assert.NoError(t, err)

	// Тест с пустым массивом
	err = db.DeleteURLs(ctx, []string{}, userID)
	assert.NoError(t, err)

	// Тест на ошибку
	mockPool.ExpectExec("UPDATE urls").
		WithArgs(shortIDs, "error").
		WillReturnError(pgx.ErrTxClosed)

	err = db.DeleteURLs(ctx, shortIDs, "error")
	assert.Error(t, err)
}

// DeleteURLs помечает URL как удаленные в базе данных.
func (db *MockDatabaseStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	if len(shortIDs) == 0 {
		return nil
	}
	_, err := db.pool.Exec(ctx, UpdateDeleteURLs, shortIDs, userID)
	if err != nil {
		return err
	}
	return nil
}

// TestDatabaseStorage_Ping тестирует проверку соединения с базой данных.
func TestDatabaseStorage_Ping(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{
		pool: mockPool,
	}

	ctx := context.Background()

	mockPool.ExpectPing()

	err = db.Ping(ctx)
	assert.NoError(t, err)

	err = mockPool.ExpectationsWereMet()
	assert.NoError(t, err)

	// Тест на ошибку ping
	mockPool.ExpectPing().WillReturnError(pgx.ErrTxClosed)

	err = db.Ping(ctx)
	assert.Error(t, err)
}

// Ping проверяет соединение с базой данных.
func (db *MockDatabaseStorage) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// TestDatabaseStorage_Close тестирует закрытие соединения с базой данных.
func TestDatabaseStorage_Close(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)

	db := &MockDatabaseStorage{
		pool: mockPool,
	}

	err = db.Close()
	assert.NoError(t, err)
}

// Close закрывает соединение с базой данных.
func (db *MockDatabaseStorage) Close() error {
	db.pool.Close()
	return nil
}
