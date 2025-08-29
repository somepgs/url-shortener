package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/somepgs/url-shortener/internal/models"
)

func TestPostgresStorage_CRUD(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("set TEST_DB_DSN to run storage integration test, e.g.: postgres://admin:password@localhost:5433/urlshortener?sslmode=disable")
	}

	store, err := NewPostgresStorage(dsn)
	if err != nil {
		t.Skipf("cannot connect to db: %v", err) // skip, чтобы не ломать CI
	}
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	u := &models.URL{
		ShortCode:   "testit",
		OriginalURL: "https://example.com",
		Clicks:      0,
		CreatedAt:   time.Now(),
	}

	if err := store.SaveURL(ctx, u); err != nil {
		t.Fatalf("SaveURL error: %v", err)
	}
	if u.ID == 0 {
		t.Fatalf("expected ID to be set after SaveURL")
	}

	got, err := store.GetURL(ctx, "testit")
	if err != nil {
		t.Fatalf("GetURL error: %v", err)
	}
	if got.OriginalURL != "https://example.com" {
		t.Fatalf("unexpected OriginalURL: %s", got.OriginalURL)
	}

	if err := store.IncrementClicks(ctx, "testit"); err != nil {
		t.Fatalf("IncrementClicks error: %v", err)
	}

	got2, err := store.GetURL(ctx, "testit")
	if err != nil {
		t.Fatalf("GetURL after increment error: %v", err)
	}
	if got2.Clicks != got.Clicks+1 {
		t.Fatalf("expected clicks to increase by 1, got %d -> %d", got.Clicks, got2.Clicks)
	}
}
