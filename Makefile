# Makefile для GophKeeper

BIN_DIR := ./bin
SERVER_BIN := $(BIN_DIR)/server
CLIENT_BIN := $(BIN_DIR)/client

.PHONY: help
help:
	@echo "Доступные команды:"
	@echo "  make certs        - Создание SSL сертификатов"
	@echo "  make build        - Собрать сервер и клиент"
	@echo "  make build-server - Собрать только сервер"
	@echo "  make build-client - Собрать только клиент"
	@echo "  make run-server   - Запустить сервер"
	@echo "  make run-client   - Запустить клиент (TUI)"
	@echo "  make test         - Запустить тесты"
	@echo "  make clean        - Очистить сборку"
	@echo "  make deps         - Установить зависимости"
	@echo "  make lint         - Проверить код"

.PHONY: all
all: build

.PHONY: build
build: build-server build-client

.PHONY: build-server
build-server:
	@echo "Сборка сервера..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(SERVER_BIN) ./cmd/server

.PHONY: build-client
build-client:
	@echo "Сборка клиента..."
	@mkdir -p $(BIN_DIR)
	@go build $(LDFLAGS) -o $(CLIENT_BIN) ./cmd/client

.PHONY: run-server
run-server: build-server
	@echo "Запуск сервера..."
	@$(SERVER_BIN)

.PHONY: run-client
run-client: build-client
	@echo "Запуск клиента..."
	@TERM=xterm-256color $(CLIENT_BIN)

.PHONY: test
test:
	@echo "Запуск тестов..."
	@go test -v ./...

.PHONY: clean
clean:
	@echo "Очистка..."
	@rm -rf $(BIN_DIR)

.PHONY: deps
deps:
	@echo "Установка зависимостей..."
	@go mod download

.PHONY: lint
lint:
	@echo "Проверка кода..."
	@golangci-lint run

.PHONY: certs
certs:
	@echo "Создание SSL сертификатов..."
	@mkdir -p ./certs
	@openssl req -x509 -newkey rsa:4096 -keyout ./certs/server.key -out ./certs/server.crt -days 365 -nodes -subj "/CN=localhost"
