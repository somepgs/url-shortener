package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/somepgs/url-shortener/internal/handlers"
	"github.com/somepgs/url-shortener/internal/service"
	"github.com/somepgs/url-shortener/internal/storage"
)

func main() {
	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Получаем DSN для БД
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN is required")
	}

	// Инициализируем storage
	store, err := storage.NewPostgresStorage(dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	defer func() {
		if cerr := store.Close(); cerr != nil {
			log.Printf("Failed to close database: %v", cerr)
		}
	}()

	// Инициализируем service
	urlService := service.NewUrlService(store)

	// Читаем базовый URL из окружения (с дефолтом для dev)
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// Инициализируем handlers (передаём baseURL)
	urlHandler := handlers.NewURLHandler(urlService, baseURL)

	// Настраиваем роутер
	r := mux.NewRouter()
	r.HandleFunc("/shorten", urlHandler.Shorten).Methods("POST")
	r.HandleFunc("/{code}", urlHandler.Redirect).Methods("GET")
	r.HandleFunc("/", homeHandler).Methods("GET")

	// Middleware для логирования
	r.Use(loggingMiddleware)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запуск сервера в отдельной горутине
	go func() {
		log.Printf("Server starting on port %s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
		<h1>URL Shortener</h1>
		<p>POST /shorten with {"url": "https://example.com"} to create short URL</p>`)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}
