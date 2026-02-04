# Используем многоэтапную сборку для уменьшения размера образа
FROM golang:1.24-alpine AS builder

# Установка необходимых пакетов
RUN apk add --no-cache git protobuf protobuf-dev

# Установка protoc-gen-go и protoc-gen-go-grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Установка рабочей директории
WORKDIR /app

# Копирование go.mod и go.sum
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование proto файлов и генерация
COPY proto/ ./proto/
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/exchange.proto

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

# Экспонируем порт HTTP
EXPOSE 8080

# Запуск приложения
CMD ["./main", "-c", "config.env"]
