# Используем многоэтапную сборку для уменьшения размера образа
FROM golang:1.24-alpine AS builder

# Установка необходимых пакетов
RUN apk add --no-cache git

# Установка рабочей директории
WORKDIR /app

# Копирование go.mod и go.sum
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd

# Финальный образ
FROM alpine:latest

# Установка сертификатов CA для HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копирование бинарника из builder
COPY --from=builder /app/main .
COPY --from=builder /app/config.env .

# Запуск приложения
CMD ["./main", "-c", "config.env"]
