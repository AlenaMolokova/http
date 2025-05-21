package database

import (
	"context"
	"fmt"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// DatabaseStorage представляет хранилище URL-адресов в PostgreSQL базе данных.
// Предоставляет методы для сохранения, поиска и удаления URL-адресов.
type DatabaseStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage создаёт и инициализирует новое хранилище PostgreSQL.
// Устанавливает соединение с базой данных по указанной строке подключения (DSN)
// и создаёт необходимые таблицы, если они ещё не существуют.
//
// Параметры:
//   - dsn: строка подключения к PostgreSQL базе данных.
//
// Возвращает:
//   - указатель на DatabaseStorage при успешной инициализации
//   - ошибку, если не удалось подключиться к базе данных или создать таблицы
func NewPostgresStorage(dsn string) (*DatabaseStorage, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	_, err = pool.Exec(context.Background(), CreateURLsTable)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create urls table: %w", err)
	}

	logrus.Info("Database storage initialized successfully")
	return &DatabaseStorage{pool: pool}, nil
}

// Save сохраняет новый URL в базе данных.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращенный идентификатор URL
//   - originalURL: оригинальный URL-адрес
//   - userID: идентификатор пользователя, который создал сокращение
//
// Возвращает:
//   - ошибку, если не удалось сохранить URL
func (db *DatabaseStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	_, err := db.pool.Exec(ctx, InsertURL, shortID, originalURL, userID)
	if err != nil {
		return fmt.Errorf("failed to save URL: %w", err)
	}
	return nil
}

// FindByOriginalURL ищет сокращенный идентификатор по оригинальному URL-адресу.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - originalURL: оригинальный URL-адрес для поиска
//
// Возвращает:
//   - сокращенный идентификатор, если URL найден
//   - пустую строку, если URL не найден
//   - ошибку, если произошла ошибка при выполнении запроса
func (db *DatabaseStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	var shortID string
	err := db.pool.QueryRow(ctx, SelectByOriginalURL, originalURL).Scan(&shortID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("failed to find URL: %w", err)
	}
	return shortID, nil
}

// Get возвращает оригинальный URL-адрес по сокращенному идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращенный идентификатор URL
//
// Возвращает:
//   - оригинальный URL-адрес и true, если сокращение найдено
//   - пустую строку и false, если сокращение не найдено или произошла ошибка
func (db *DatabaseStorage) Get(ctx context.Context, shortID string) (string, bool) {
	var originalURL string
	err := db.pool.QueryRow(ctx, SelectByShortID, shortID).Scan(&originalURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false
		}
		logrus.WithError(err).Error("Failed to get URL")
		return "", false
	}
	return originalURL, true
}

// GetURLsByUserID возвращает все URL-адреса, созданные указанным пользователем.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - список структур UserURL, содержащих сокращенные и оригинальные URL-адреса
//   - ошибку, если произошла ошибка при выполнении запроса
func (db *DatabaseStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	rows, err := db.pool.Query(ctx, SelectByUserID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query URLs: %w", err)
	}
	defer rows.Close()

	var urls []models.UserURL
	for rows.Next() {
		var shortID, originalURL, userID string
		var isDeleted bool
		if err := rows.Scan(&shortID, &originalURL, &userID, &isDeleted); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		urls = append(urls, models.UserURL{ShortURL: shortID, OriginalURL: originalURL})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return urls, nil
}

// SaveBatch сохраняет пакет URL-адресов в базе данных в рамках одной транзакции.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - batch: карта, где ключ - сокращенный идентификатор, значение - оригинальный URL
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - ошибку, если не удалось сохранить пакет URL-адресов
func (db *DatabaseStorage) SaveBatch(ctx context.Context, batch map[string]string, userID string) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for shortID, originalURL := range batch {
		_, err := tx.Exec(ctx, InsertURLBatch, shortID, originalURL, userID)
		if err != nil {
			return fmt.Errorf("failed to save batch URL: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// DeleteURLs помечает указанные URL-адреса как удаленные.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortIDs: список сокращенных идентификаторов для удаления
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - ошибку, если не удалось пометить URL-адреса как удаленные
func (db *DatabaseStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	if len(shortIDs) == 0 {
		return nil
	}
	_, err := db.pool.Exec(ctx, UpdateDeleteURLs, shortIDs, userID)
	if err != nil {
		return fmt.Errorf("failed to delete URLs: %w", err)
	}
	return nil
}

// Ping проверяет доступность базы данных.
//
// Параметры:
//   - ctx: контекст выполнения операции
//
// Возвращает:
//   - ошибку, если база данных недоступна
func (db *DatabaseStorage) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close закрывает соединение с базой данных.
//
// Возвращает:
//   - ошибку, если не удалось корректно закрыть соединение
func (db *DatabaseStorage) Close() error {
	db.pool.Close()
	return nil
}
