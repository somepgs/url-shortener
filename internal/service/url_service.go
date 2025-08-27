package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/somepgs/url-shortener/internal/models"
	"github.com/somepgs/url-shortener/internal/storage"
)

type UrlService struct {
	storage storage.Storage
}

func NewUrlService(storage storage.Storage) *UrlService {
	return &UrlService{storage: storage}
}

func (s *UrlService) ShortenURL(ctx context.Context, originalURL string) (*models.URL, error) {
	// Пытаемся несколько раз в случае коллизии короткого кода
	const maxAttempts = 5
	var lastErr error

	for i := 0; i < maxAttempts; i++ {
		shortCode := generateShortCode()

		url := &models.URL{
			ShortCode:   shortCode,
			OriginalURL: originalURL,
			Clicks:      0,
			CreatedAt:   time.Now(),
		}

		if err := s.storage.SaveURL(ctx, url); err != nil {
			// Если уникальный индекс сработал - пробуем другой код
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				lastErr = err
				continue
			}
			return nil, err
		}

		return url, nil
	}

	return nil, lastErr
}

func (s *UrlService) GetAndRedirect(ctx context.Context, shortCode string) (string, error) {
	url, err := s.storage.GetURL(ctx, shortCode)
	if err != nil {
		return "", err
	}

	// Асинхронно увеличиваем счетчик
	go s.storage.IncrementClicks(context.Background(), shortCode)

	return url.OriginalURL, nil
}

func generateShortCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	str := base64.RawURLEncoding.EncodeToString(b)
	//	Убираем символы, которые могут вызвать проблемы в URL
	str = strings.ReplaceAll(str, "+", "-")
	str = strings.ReplaceAll(str, "/", "_")
	if len(str) < 6 {
		return str + "0"[:6-len(str)]
	}
	return str[:6]
}
