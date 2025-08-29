# URL Shortener

Простое и аккуратное сервис-приложение для сокращения ссылок на Go с хранением в PostgreSQL.

## Возможности
- Генерация коротких кодов и редирект на оригинальный URL
- Подсчет кликов (инкремент при каждом переходе)
- Разделение на слои: handlers → service → storage
- Конфигурация через переменные окружения (.env)
- Тесты для сервиса и HTTP-обработчиков

## Технологии
- Go 1.25
- Gorilla Mux (HTTP роутер)
- PostgreSQL + драйвер lib/pq
- Docker Compose (для локального запуска БД)
- godotenv (загрузка .env)

## Быстрый старт

### 1) Клонирование
```bash
git clone https://github.com/USER/url-shortener.git
cd url-shortener
```
### 2) Конфигурация
Создайте файл `.env` на основе примера:
Заполните значения:
- `DB_DSN` — строка подключения к PostgreSQL, например:
    - `postgres://admin:password@localhost:5433/urlshortener?sslmode=disable`

- `PORT` — порт HTTP-сервера (по умолчанию 8080)
- `BASE_URL` — базовый адрес для формирования коротких ссылок, например:
    - `http://localhost:8080`

Файл `.env` добавлен в `.gitignore` и не должен попадать в репозиторий.
### 3) Поднять базу данных (Docker Compose)
```bash
docker compose up -d
```
По умолчанию БД доступна на `localhost:5433` (снаружи) и `5432` (внутри контейнера).
Инициализация схемы:
- При первом старте контейнера Postgres автоматически выполнит SQL-скрипты из папки `./migrations` (монтируется в `/docker-entrypoint-initdb.d`).
  Если вы меняли миграции после первого старта, пересоздайте volume:
```bash
docker compose down -v
docker compose up -d
```
или примените миграции вручную через `psql`.
### 4) Запуск приложения
```bash
go run cmd/api/main.go
```
Сервер поднимется на `http://localhost:${PORT}` (по умолчанию `http://localhost:8080`).


## API
### POST /shorten
Создать короткую ссылку.
Запрос:
```
POST /shorten
Content-Type: application/json

{
  "url": "https://example.com"
}
```
Успешный ответ (201):
```json
{
    "short_url": "http://localhost:8080/abc123",
    "original_url": "https://example.com"
}
```
Ошибки:
- 400 Invalid request (невалидный JSON)
- 400 URL is required (пустой URL)
- 500 (ошибка сервера/БД)

### GET /{code}
Редирект по короткому коду.
Пример:
```
GET /abc123
```
Ответ:
- 307 (Temporary Redirect) c заголовком `Location: https://example.com`
- 404 (если код не найден)


## Переменные окружения
- `DB_DSN` (обязательно) — строка подключения к БД.
    - Локально через Docker Compose: `postgres://admin:password@localhost:5433/urlshortener?sslmode=disable`
- `PORT` (опционально) — порт HTTP-сервера (дефолт: `8080`).
- `BASE_URL` (опционально, но рекомендуется) — базовый URL для short_url в ответе (например, `http://localhost:8080`).
    - Это важно для разных окружений (staging/prod), прокси и HTTPS.


## Тесты
Запуск всех тестов:
```bash
go test ./... -race -cover
```
Интеграционный тест хранилища (опционально):
```bash
export TEST_DB_DSN="postgres://admin:password@localhost:5433/urlshortener?sslmode=disable"
go test ./internal/storage -run TestPostgresStorage_CRUD -race -v
```
Отчет покрытия в HTML:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```
