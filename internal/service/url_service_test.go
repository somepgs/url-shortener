package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/somepgs/url-shortener/internal/models"
)

// fakeStorage реализует storage.Storage для юнит-тестов сервиса.
type fakeStorage struct {
	saveCalls         int
	saveErrSequence   []error
	urls              map[string]*models.URL
	incrementCalls    int
	incrementLastCode string
	incrementCalledCh chan struct{}
}

func (f *fakeStorage) SaveURL(ctx context.Context, u *models.URL) error {
	f.saveCalls++
	if f.urls == nil {
		f.urls = make(map[string]*models.URL)
	}

	if len(f.saveErrSequence) >= f.saveCalls {
		return f.saveErrSequence[f.saveCalls-1]
	}
	f.urls[u.ShortCode] = u

	u.ID = f.saveCalls
	return nil
}

func (f *fakeStorage) GetURL(ctx context.Context, code string) (*models.URL, error) {
	u, ok := f.urls[code]
	if !ok {
		return nil, errors.New("url not found")
	}
	return u, nil
}

func (f *fakeStorage) IncrementClicks(ctx context.Context, code string) error {
	f.incrementCalls++
	f.incrementLastCode = code
	if f.urls != nil {
		if u, ok := f.urls[code]; ok {
			u.Clicks++
		}
	}

	// Сигнализируем, что асинхронный вызов состоялся
	if f.incrementCalledCh != nil {
		select {
		case f.incrementCalledCh <- struct{}{}:
		default:
		}
	}
	return nil
}

func TestShortenURL_Success(t *testing.T) {
	st := &fakeStorage{}
	svc := NewUrlService(st)

	u, err := svc.ShortenURL(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ShortCode == "" {
		t.Fatalf("expected short code to be generated")
	}
	if u.Clicks != 0 {
		t.Fatalf("expected clicks to be 0, got %d", u.Clicks)
	}
	if time.Since(u.CreatedAt) > time.Second {
		t.Fatalf("expected recent CreatedAt, got %v", u.CreatedAt)
	}
}

func TestShortenURL_RetryOnUniqueViolation(t *testing.T) {
	st := &fakeStorage{
		saveErrSequence: []error{
			&pq.Error{Code: "23505"},
		},
	}
	svc := NewUrlService(st)

	u, err := svc.ShortenURL(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error after retry: %v", err)
	}
	if st.saveCalls < 2 {
		t.Fatalf("expected at least 2 save attempts, got %d", st.saveCalls)
	}
	if u.ShortCode == "" {
		t.Fatalf("expected short code to be generated")
	}
}

func TestGetAndRedirect_Success_IncrementsClicksAsync(t *testing.T) {
	ch := make(chan struct{}, 1)
	st := &fakeStorage{
		urls: map[string]*models.URL{
			"abc123": {ShortCode: "abc123", OriginalURL: "https://example.com"},
		},
		incrementCalledCh: ch,
	}
	svc := NewUrlService(st)

	url, err := svc.GetAndRedirect(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com" {
		t.Fatalf("unexpected redirect url: %s", url)
	}

	// Ждём асинхронный вызов IncrementClicks (не более 500мс).
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected IncrementClicks to be called asynchronously")
	}

	if st.incrementCalls == 0 || st.incrementLastCode != "abc123" {
		t.Fatalf("expected IncrementClicks to be called for code abc123")
	}
}

func TestGetAndRedirect_NotFound(t *testing.T) {
	st := &fakeStorage{urls: map[string]*models.URL{}}
	svc := NewUrlService(st)

	_, err := svc.GetAndRedirect(context.Background(), "nope")
	if err == nil {
		t.Fatalf("expected error for not found")
	}
}
