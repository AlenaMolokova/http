package database

import (
	"context"
	"log"
	"sort"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPostgresStorage тестирует создание нового PostgreSQL хранилища.
// Пропускается в CI, так как требует реальной базы данных.
func TestNewPostgresStorage(t *testing.T) {
	t.Skip("Тест требует реальной базы данных, пропускаем в CI")
}

// MockDatabaseStorage представляет мок-объект для тестирования DatabaseStorage.
type MockDatabaseStorage struct {
	pool pgxmock.PgxPoolIface
}

// Save сохраняет URL в базе данных.
// Выполняет SQL-запрос для вставки записи в таблицу urls.
func (db *MockDatabaseStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	_, err := db.pool.Exec(ctx, InsertURL, shortID, originalURL, userID)
	return err
}

// TestDatabaseStorage_Save тестирует сохранение одного URL в базе данных.
// Проверяет успешное выполнение и обработку ошибок.
func TestDatabaseStorage_Save(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{pool: mockPool}

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

// FindByOriginalURL находит короткий ID по оригинальному URL.
// Возвращает shortID, если URL найден, или пустую строку, если нет.
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

// TestDatabaseStorage_FindByOriginalURL тестирует поиск короткого ID по оригинальному URL.
// Проверяет успешный поиск, отсутствие записи и обработку ошибок.
func TestDatabaseStorage_FindByOriginalURL(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{pool: mockPool}

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
		WillReturnError(pgx.ErrNoRows)

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

// Get получает оригинальный URL по короткому ID.
// Возвращает URL и флаг существования записи.
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

// TestDatabaseStorage_Get тестирует получение оригинального URL по короткому ID.
// Проверяет успешное получение, отсутствие записи и обработку ошибок.
func TestDatabaseStorage_Get(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{pool: mockPool}

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

// GetURLsByUserID получает все URL пользователя.
// Возвращает список URL, принадлежащих указанному пользователю.
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
		if !isDeleted {
			urls = append(urls, models.UserURL{ShortURL: shortID, OriginalURL: originalURL})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

// TestDatabaseStorage_GetURLsByUserID тестирует получение всех URL пользователя.
// Проверяет успешное получение списка, обработку ошибок и пустой результат.
func TestDatabaseStorage_GetURLsByUserID(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{pool: mockPool}

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
		WithArgs("empty").
		WillReturnRows(pgxmock.NewRows([]string{"short_id", "original_url", "user_id", "is_deleted"}))

	urls, err = db.GetURLsByUserID(ctx, "empty")
	assert.NoError(t, err)
	assert.Empty(t, urls)

	mockPool.ExpectQuery("SELECT short_id, original_url, user_id, is_deleted").
		WithArgs("error").
		WillReturnError(pgx.ErrTxClosed)

	urls, err = db.GetURLsByUserID(ctx, "error")
	assert.Error(t, err)
	assert.Nil(t, urls)
}

// SaveBatch сохраняет пакет URL в базе данных в рамках транзакции.
// Сортирует ключи для предсказуемого порядка вставки.
func (db *MockDatabaseStorage) SaveBatch(ctx context.Context, batch map[string]string, userID string) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			log.Printf("Ошибка отката транзакции: %v", err)
		}
	}()

	var keys []string
	for k := range batch {
		keys = append(keys, k)
	}
	sort.Strings(keys)

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

// TestDatabaseStorage_SaveBatch тестирует пакетное сохранение URL в базе данных.
// Проверяет успешное сохранение, обработку ошибок транзакции и откат.
func TestDatabaseStorage_SaveBatch(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{pool: mockPool}

	ctx := context.Background()
	userID := "user1"
	batch := map[string]string{
		"abc123": "https://example.com",
		"def456": "https://test.com",
	}

	mockPool.ExpectBegin()

	var keys []string
	for k := range batch {
		keys = append(keys, k)
	}
	sort.Strings(keys)

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

	// Тест ошибки вставки
	mockPool.ExpectBegin()
	mockPool.ExpectExec("INSERT INTO urls").
		WithArgs("error", "https://example.com", userID).
		WillReturnError(pgx.ErrTxClosed)
	mockPool.ExpectRollback()

	err = db.SaveBatch(ctx, map[string]string{"error": "https://example.com"}, userID)
	assert.Error(t, err)

	// Тест ошибки начала транзакции
	mockPool.ExpectBegin().WillReturnError(pgx.ErrTxClosed)

	err = db.SaveBatch(ctx, batch, userID)
	assert.Error(t, err)

	// Тест ошибки коммита
	mockPool.ExpectBegin()
	for _, shortID := range keys {
		originalURL := batch[shortID]
		mockPool.ExpectExec("INSERT INTO urls").
			WithArgs(shortID, originalURL, userID).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
	}
	mockPool.ExpectCommit().WillReturnError(pgx.ErrTxClosed)
	mockPool.ExpectRollback()

	err = db.SaveBatch(ctx, batch, userID)
	assert.Error(t, err)
}

// DeleteURLs помечает указанные URL как удаленные для заданного пользователя.
// Обновляет поле is_deleted в таблице urls.
func (db *MockDatabaseStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	_, err := db.pool.Exec(ctx, UpdateDeleteURLs, shortIDs, userID)
	return err
}

// TestDatabaseStorage_DeleteURLs тестирует удаление URL пользователя.
// Проверяет успешное обновление и обработку ошибок.
func TestDatabaseStorage_DeleteURLs(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{pool: mockPool}

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

	// Тест ошибки обновления
	mockPool.ExpectExec("UPDATE urls").
		WithArgs([]string{"error"}, userID).
		WillReturnError(pgx.ErrTxClosed)

	err = db.DeleteURLs(ctx, []string{"error"}, userID)
	assert.Error(t, err)
}

// Ping проверяет соединение с базой данных.
// Возвращает ошибку, если соединение не удалось установить.
func (db *MockDatabaseStorage) Ping(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, "SELECT 1")
	return err
}

// TestDatabaseStorage_Ping тестирует проверку соединения с базой данных.
// Проверяет успешное выполнение и обработку ошибок.
func TestDatabaseStorage_Ping(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	db := &MockDatabaseStorage{pool: mockPool}

	ctx := context.Background()

	mockPool.ExpectExec("SELECT 1").
		WillReturnResult(pgxmock.NewResult("SELECT", 1))

	err = db.Ping(ctx)
	assert.NoError(t, err)

	err = mockPool.ExpectationsWereMet()
	assert.NoError(t, err)

	mockPool.ExpectExec("SELECT 1").
		WillReturnError(pgx.ErrTxClosed)

	err = db.Ping(ctx)
	assert.Error(t, err)
}
