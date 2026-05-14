.PHONY: build run clean test test-unit test-integration test-coverage test-with-script start-network stop-network lint profile dev-env docker-build docker-run docker-clean web install-deps help

build:
	go build -o bin/aetherwave cmd/aetherwave/main.go

run:
	go run cmd/aetherwave/main.go $(ARGS)

clean:
	rm -rf bin/ coverage/ test-logs/ .pid/
	@echo "Очистка выполнена успешно"

test:
	mkdir -p test-logs
	go test ./... -v 2>&1 | tee test-logs/all_tests.log

test-unit:
	mkdir -p test-logs
	go test ./pkg/... -v 2>&1 | tee test-logs/unit_tests.log

test-integration:
	mkdir -p test-logs
	go test ./tests/... -v 2>&1 | tee test-logs/integration_tests.log

test-coverage:
	mkdir -p coverage test-logs
	go test ./... -coverprofile=coverage/coverage.out -covermode=atomic -v 2>&1 | tee test-logs/coverage_tests.log
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated at coverage/coverage.html"

test-coverage-unit:
	mkdir -p coverage test-logs
	go test ./pkg/... -coverprofile=coverage/unit_coverage.out -covermode=atomic -v 2>&1 | tee test-logs/unit_coverage_tests.log
	go tool cover -html=coverage/unit_coverage.out -o coverage/unit_coverage.html
	@echo "Unit test coverage report generated at coverage/unit_coverage.html"

test-coverage-integration:
	mkdir -p coverage test-logs
	go test ./tests/... -coverprofile=coverage/integration_coverage.out -covermode=atomic -v 2>&1 | tee test-logs/integration_coverage_tests.log
	go tool cover -html=coverage/integration_coverage.out -o coverage/integration_coverage.html
	@echo "Integration test coverage report generated at coverage/integration_coverage.html"

test-with-script:
	./scripts/run_tests.sh

# Новые цели

start-network:
	@echo "Запуск сети AetherWave..."
	./scripts/start_network.sh
	@echo "Сеть AetherWave запущена"

stop-network:
	@echo "Остановка сети AetherWave..."
	./scripts/stop_network.sh
	@echo "Сеть AetherWave остановлена"

lint:
	@echo "Запуск линтера..."
	go vet ./cmd/... ./pkg/... ./sdk/... ./tests/... ./mobile/...
	@if command -v golint > /dev/null; then \
		golint ./...; \
	else \
		echo "golint не установлен. Установите его с помощью: go install golang.org/x/lint/golint@latest"; \
	fi
	@if command -v staticcheck > /dev/null; then \
		staticcheck ./...; \
	else \
		echo "staticcheck не установлен. Установите его с помощью: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi
	@echo "Линтинг завершен"

profile:
	@echo "Запуск профилирования производительности..."
	mkdir -p profiles
	go run scripts/profile_performance.go -type cpu -output profiles/cpu_profile.out -messages 1000 -duration 30s
	@echo "Профилирование CPU завершено. Результаты в profiles/cpu_profile.out"
	go run scripts/profile_performance.go -type memory -output profiles/mem_profile.out -messages 1000
	@echo "Профилирование памяти завершено. Результаты в profiles/mem_profile.out"
	go run scripts/profile_performance.go -type block -output profiles/block_profile.out -messages 10 -difficulty 3
	@echo "Профилирование создания блоков завершено. Результаты в profiles/block_profile.out"

dev-env:
	@echo "Настройка среды разработки..."
	@if ! command -v golint > /dev/null; then \
		echo "Установка golint..."; \
		go install golang.org/x/lint/golint@latest; \
	fi
	@if ! command -v staticcheck > /dev/null; then \
		echo "Установка staticcheck..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi
	@if ! command -v go-critic > /dev/null; then \
		echo "Установка go-critic..."; \
		go install github.com/go-critic/go-critic/cmd/gocritic@latest; \
	fi
	@echo "Среда разработки настроена"

docker-build:
	@echo "Сборка Docker-образа..."
	docker build -t aetherwave:latest .
	@echo "Docker-образ успешно собран"

docker-run:
	@echo "Запуск Docker-контейнеров..."
	docker compose up -d 2>/dev/null || docker-compose up -d
	@echo "Docker-контейнеры запущены"

docker-clean:
	@echo "Остановка и удаление Docker-контейнеров..."
	docker compose down 2>/dev/null || docker-compose down
	@echo "Docker-контейнеры остановлены и удалены"

web:
	@echo "Запуск веб-интерфейса..."
	open web/index.html || xdg-open web/index.html || echo "Не удалось автоматически открыть веб-интерфейс. Откройте web/index.html вручную."

install-deps:
	@echo "Установка зависимостей..."
	go mod tidy
	go mod download
	@echo "Зависимости установлены"

help:
	@echo "AetherWave Blockchain Makefile"
	@echo "Доступные команды:"
	@echo "  make build                 - Сборка приложения"
	@echo "  make run                   - Запуск приложения"
	@echo "  make clean                 - Очистка временных файлов"
	@echo "  make test                  - Запуск всех тестов"
	@echo "  make test-unit             - Запуск только юнит-тестов"
	@echo "  make test-integration      - Запуск только интеграционных тестов"
	@echo "  make test-coverage         - Запуск тестов с отчетом о покрытии"
	@echo "  make start-network         - Запуск сети AetherWave"
	@echo "  make stop-network          - Остановка сети AetherWave"
	@echo "  make lint                  - Запуск линтера"
	@echo "  make profile               - Запуск профилирования"
	@echo "  make dev-env               - Настройка среды разработки"
	@echo "  make docker-build          - Сборка Docker-образа"
	@echo "  make docker-run            - Запуск Docker-контейнеров"
	@echo "  make docker-clean          - Остановка и удаление Docker-контейнеров"
	@echo "  make web                   - Запуск веб-интерфейса"
	@echo "  make install-deps          - Установка зависимостей"
	@echo "  make help                  - Показать эту справку"

.DEFAULT_GOAL := help