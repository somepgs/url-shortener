package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/somepgs/url-shortener/internal/models"
	"github.com/somepgs/url-shortener/internal/service"
)

// fakeStorage реализует storage.Storage для тестирования HTTP-обработчиков через реальный сервис.
type fakeStorage struct {
	urls    map[string]*models.URL
	saveErr error
}

func (f *fakeStorage) SaveURL(ctx context.Context, u *models.URL) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.urls == nil {
		f.urls = make(map[string]*models.URL)
	}
	// Присвоим ID и сохраним
	u.ID = len(f.urls) + 1
	f.urls[u.ShortCode] = u
	return nil
}

func (f *fakeStorage) GetURL(ctx context.Context, code string) (*models.URL, error) {
	if f.urls == nil {
		return nil, errors.New("url not found")
	}
	u, ok := f.urls[code]
	if !ok {
		return nil, errors.New("url not found")
	}
	return u, nil
}

func (f *fakeStorage) IncrementClicks(ctx context.Context, code string) error {
	if f.urls == nil {
		return nil
	}
	if u, ok := f.urls[code]; ok {
		u.Clicks++
	}
	return nil
}

func setupRouter(h *URLHandler) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/shorten", h.Shorten).Methods("POST")
	r.HandleFunc("/{code}", h.Redirect).Methods("GET")
	return r
}

func TestShorten_Success(t *testing.T) {
	st := &fakeStorage{urls: make(map[string]*models.URL)}
	svc := service.NewUrlService(st)
	const baseURL = "http://localhost:8080"
	h := NewURLHandler(svc, baseURL)
	r := setupRouter(h)

	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var resp struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if resp.OriginalURL != "https://example.com" {
		t.Fatalf("unexpected original_url: %s", resp.OriginalURL)
	}
	if !strings.HasPrefix(resp.ShortURL, baseURL+"/") {
		t.Fatalf("short_url should start with %q, got %q", baseURL+"/", resp.ShortURL)
	}
	if len(resp.ShortURL) <= len(baseURL)+1 {
		t.Fatalf("short_url should contain code after baseURL, got %q", resp.ShortURL)
	}
}

func TestShorten_BadJSON(t *testing.T) {
	st := &fakeStorage{}
	svc := service.NewUrlService(st)
	h := NewURLHandler(svc, "http://localhost:8080")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad json, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &e); err != nil {
		t.Fatalf("invalid error json: %v", err)
	}
	if e.Error != "Invalid request" {
		t.Fatalf("unexpected error message: %q", e.Error)
	}
}

func TestShorten_EmptyURL(t *testing.T) {
	st := &fakeStorage{}
	svc := service.NewUrlService(st)
	h := NewURLHandler(svc, "http://localhost:8080")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewBufferString(`{"url":""}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty url, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &e); err != nil {
		t.Fatalf("invalid error json: %v", err)
	}
	if e.Error != "URL is required" {
		t.Fatalf("unexpected error message: %q", e.Error)
	}
}

func TestRedirect_Found(t *testing.T) {
	st := &fakeStorage{
		urls: map[string]*models.URL{
			"abc123": {
				ID:          1,
				ShortCode:   "abc123",
				OriginalURL: "https://example.com/page",
				Clicks:      0,
				CreatedAt:   time.Now(),
			},
		},
	}
	svc := service.NewUrlService(st)
	h := NewURLHandler(svc, "http://localhost:8080")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if loc != "https://example.com/page" {
		t.Fatalf("expected Location header to be original url, got %q", loc)
	}
}

func TestRedirect_NotFound(t *testing.T) {
	st := &fakeStorage{urls: map[string]*models.URL{}}
	svc := service.NewUrlService(st)
	h := NewURLHandler(svc, "http://localhost:8080")
	r := setupRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
