FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем только файлы зависимостей для кеширования слоев
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/aetherwave cmd/aetherwave/main.go

# Финальный образ
FROM alpine:3.20

RUN adduser -D -h /app aetherwave

WORKDIR /app

# Копируем бинарный файл из предыдущего этапа
COPY --from=builder /app/bin/aetherwave .

# Создаем директории для данных
RUN mkdir -p /app/data /app/logs && chown -R aetherwave:aetherwave /app

USER aetherwave

# Открываем порты
EXPOSE 3000 8080

# Запускаем приложение
ENTRYPOINT ["/app/aetherwave"]
CMD ["--address", ":3000"] 