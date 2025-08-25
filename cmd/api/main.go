package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Временное хранилище в памяти (потом заменим на БД)
type Storage struct {
	urls map[string]string
	mu   sync.RWMutex
}

func NewStorage() *Storage {
	return &Storage{
		urls: make(map[string]string),
	}
}

// Структура запроса
type ShortenRequest struct {
	URL string `json:"url"`
}

// Структура ответа
type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

var storage = NewStorage()

func main() {
	// Регистрируем хендлеры
	http.HandleFunc("/shorten", shortenHandler)
	http.HandleFunc("/", redirectHandler)

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Хендлер для создания короткой ссылки
func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Простая валидация
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Генерируем короткий код (пока просто MD5 и берем первые 6 символов)
	hash := md5.Sum([]byte(req.URL))
	shortCode := hex.EncodeToString(hash[:])[:6]

	// Сохраняем
	storage.mu.Lock()
	storage.urls[shortCode] = req.URL
	storage.mu.Unlock()

	// Отправляем ответ
	resp := ShortenResponse{
		ShortURL: fmt.Sprintf("http://localhost:8080/%s", shortCode),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Хендлер для редиректа
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		fmt.Fprint(w, "URL Shortener Service")
		return
	}

	// Получаем код из URL
	shortCode := r.URL.Path[1:] // убираем первый слэш

	// Ищем оригинальный URL
	storage.mu.RLock()
	originalURL, exists := storage.urls[shortCode]
	storage.mu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	// Редиректим
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}
