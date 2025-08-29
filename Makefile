SHELL := /bin/bash

GO       := go
APP_PKG  := cmd/api/main.go
BIN_DIR  := bin
BIN_NAME := url-shortener

.PHONY: run build test cover cover-html lint compose-up compose-down db-psql init-env clean

# Запуск приложения (читает .env, если используете godotenv)
run:
	$(GO) run $(APP_PKG)

# Сборка бинарника
build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BIN_NAME) $(APP_PKG)

# Юнит- и хендлер-тесты
test:
	$(GO) test ./... -race -cover

# Покрытие в текстовом виде
cover:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

# Покрытие в HTML
cover-html:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out

# Линт (нужен golangci-lint в PATH)
lint:
	golangci-lint run

# Поднять БД и прочие сервисы
compose-up:
	docker compose up -d

# Остановить compose-сервисы
compose-down:
	docker compose down

# Быстрый вход в psql по TEST_DB_DSN
# Пример: make db-psql TEST_DB_DSN="postgres://admin:password@localhost:5433/urlshortener?sslmode=disable"
db-psql:
	psql "$${TEST_DB_DSN}"

# Создать .env из .env.example (если .env отсутствует)
init-env:
	@test -f .env || { cp .env.example .env && echo ".env created from .env.example"; }

# Очистить артефакты сборки/покрытия
clean:
	rm -rf $(BIN_DIR) coverage.out
