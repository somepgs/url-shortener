package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/somepgs/url-shortener/internal/models"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Параметры пула соединений
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(60 * time.Minute)
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)

	// Пинг с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

func (s *PostgresStorage) SaveURL(ctx context.Context, url *models.URL) error {
	query := `
		INSERT INTO urls (short_code, original_url, clicks, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err := s.db.QueryRowContext(
		ctx, query,
		url.ShortCode, url.OriginalURL, url.Clicks, url.CreatedAt,
	).Scan(&url.ID)

	return err
}

func (s *PostgresStorage) GetURL(ctx context.Context, shortCode string) (*models.URL, error) {
	var url models.URL
	query := `
		SELECT id, short_code, original_url, clicks, created_at
		FROM urls
		WHERE short_code = $1`

	err := s.db.QueryRowContext(
		ctx, query,
		shortCode,
	).Scan(&url.ID, &url.ShortCode, &url.OriginalURL, &url.Clicks, &url.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("url not found")
	}

	return &url, err
}

func (s *PostgresStorage) IncrementClicks(ctx context.Context, shortCode string) error {
	query := `
		UPDATE urls
		SET clicks = clicks + 1
		WHERE short_code = $1`
	_, err := s.db.ExecContext(ctx, query, shortCode)
	return err
}
