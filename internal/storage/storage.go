package storage

import (
	"context"

	"github.com/somepgs/url-shortener/internal/models"
)

type Storage interface {
	SaveURL(ctx context.Context, url *models.URL) error
	GetURL(ctx context.Context, shortCode string) (*models.URL, error)
	IncrementClicks(ctx context.Context, shortCode string) error
}
