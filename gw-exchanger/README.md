# GW-Exchanger Service

gRPC-сервис для получения курсов валют из PostgreSQL базы данных.

## Возможности

- Получение всех курсов обмена валют
- Получение курса для конкретной пары валют
- Поддержка валют: USD, EUR, RUB
- Продвинутое логирование (JSON формат)
- Graceful shutdown
- Интерфейс для легкой замены БД

## Структура проекта

```
gw-exchanger/
├── cmd/
│   └── main.go                 # Точка входа приложения
├── pkg/
│   └── utils.go                # Утилиты
├── internal/
│   ├── storages/
│   │   ├── storage.go          # Интерфейс хранилища
│   │   ├── model.go            # Модели данных
│   │   └── postgres/
│   │       ├── connector.go    # Подключение к PostgreSQL
│   │       └── methods.go      # Методы работы с БД
│   ├── config/
│   │   ├── config.go           # Загрузка конфигурации
│   │   └── defaults.go         # Значения по умолчанию
│   ├── grpc/
│   │   └── server.go           # gRPC сервер
│   └── logger/
│       └── logger.go           # Настройка логгера
├── go.mod
├── Dockerfile
├── config.env                   # Конфигурация окружения
└── README.md
```

## Требования

- Go 1.21+
- PostgreSQL 12+
- protoc (Protocol Buffers compiler)

## Установка

### 1. Клонирование репозитория

```bash
cd gpt/gw-exchanger
```

### 2. Установка зависимостей

```bash
go mod download
```

### 3. Генерация proto файлов

Сначала убедитесь, что proto файлы сгенерированы:

```bash
cd ../proto-exchange
./generate.sh
cd ../gw-exchanger
```

### 4. Настройка базы данных

Создайте PostgreSQL базу данных:

```sql
CREATE DATABASE exchanger_db;
CREATE USER exchanger_user WITH PASSWORD 'exchanger_password';
GRANT ALL PRIVILEGES ON DATABASE exchanger_db TO exchanger_user;
```

### 5. Настройка конфигурации

Отредактируйте `config.env` при необходимости:

```env
GRPC_PORT=50051
LOG_LEVEL=info

DB_HOST=localhost
DB_PORT=5432
DB_USER=exchanger_user
DB_PASSWORD=exchanger_password
DB_NAME=exchanger_db
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

## Запуск

### Локальный запуск

```bash
# Сборка
GOOS=linux GOARCH=amd64 go build -o main ./cmd

# Запуск
./main -c config.env
```

### Docker запуск

```bash
# Сборка образа
docker build -t gw-exchanger .

# Запуск контейнера
docker run -p 50051:50051 --env-file config.env gw-exchanger
```

### Docker Compose

Создайте `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_USER: exchanger_user
      POSTGRES_PASSWORD: exchanger_password
      POSTGRES_DB: exchanger_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  exchanger:
    build: .
    ports:
      - "50051:50051"
    depends_on:
      - postgres
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: exchanger_user
      DB_PASSWORD: exchanger_password
      DB_NAME: exchanger_db
      GRPC_PORT: 50051
      LOG_LEVEL: info

volumes:
  postgres_data:
```

Запуск:

```bash
docker-compose up -d
```

## API

### gRPC методы

#### GetExchangeRates

Получить все курсы обмена валют.

**Запрос:**
```protobuf
message Empty {}
```

**Ответ:**
```protobuf
message ExchangeRatesResponse {
    map<string, float> rates = 1;
}
```

**Пример использования (grpcurl):**
```bash
grpcurl -plaintext localhost:50051 exchange.ExchangeService/GetExchangeRates
```

#### GetExchangeRateForCurrency

Получить курс для конкретной пары валют.

**Запрос:**
```protobuf
message CurrencyRequest {
    string from_currency = 1;
    string to_currency = 2;
}
```

**Ответ:**
```protobuf
message ExchangeRateResponse {
    string from_currency = 1;
    string to_currency = 2;
    float rate = 3;
}
```

**Пример использования (grpcurl):**
```bash
grpcurl -plaintext -d '{"from_currency":"USD","to_currency":"EUR"}' \
  localhost:50051 exchange.ExchangeService/GetExchangeRateForCurrency
```

## Логирование

Сервис использует структурированное логирование в формате JSON:

```json
{
  "timestamp": "2024-02-02 15:04:05",
  "level": "info",
  "message": "gRPC method: /exchange.ExchangeService/GetExchangeRates, duration: 5ms, status: success"
}
```

Уровни логирования: `debug`, `info`, `warn`, `error`

## Начальные данные

При первом запуске автоматически создаются:

**Валюты:**
- USD - US Dollar
- EUR - Euro
- RUB - Russian Ruble

**Курсы обмена:**
- USD -> EUR: 0.92
- USD -> RUB: 92.50
- EUR -> USD: 1.09
- EUR -> RUB: 100.54
- RUB -> USD: 0.0108
- RUB -> EUR: 0.0099

## Расширение

Для добавления поддержки другой БД:

1. Создайте новый пакет в `internal/storages/` (например, `mongodb/`)
2. Реализуйте интерфейс `Storage`
3. Обновите `cmd/main.go` для использования новой реализации

## Лицензия

MIT
