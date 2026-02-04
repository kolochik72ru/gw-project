# GW-Currency-Wallet Service

HTTP REST API сервис для управления кошельком с поддержкой обмена валют, JWT авторизации и уведомлений через Kafka.


## Поддерживаемые валюты

- USD (US Dollar)
- EUR (Euro)
- RUB (Russian Ruble)

## Структура проекта

```
gw-currency-wallet/
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
│   │       ├── methods.go      # Методы работы с пользователями
│   │       └── transactions.go # Методы работы с транзакциями
│   ├── config/
│   │   ├── config.go           # Загрузка конфигурации
│   │   └── defaults.go         # Значения по умолчанию
│   ├── api/
│   │   ├── handlers/           # HTTP обработчики
│   │   │   ├── auth.go         # Регистрация/авторизация
│   │   │   ├── wallet.go       # Операции с кошельком
│   │   │   └── exchange.go     # Обмен валют
│   │   ├── middleware/
│   │   │   ├── jwt.go          # JWT авторизация
│   │   │   └── logger.go       # Логирование запросов
│   │   └── router.go           # Настройка маршрутов
│   ├── grpc/
│   │   └── client.go           # gRPC клиент для exchanger
│   ├── cache/
│   │   └── rates_cache.go      # Кеш курсов валют
│   ├── kafka/
│   │   └── producer.go         # Kafka producer
│   ├── service/
│   │   └── wallet_service.go   # Бизнес-логика
│   └── logger/
│       └── logger.go           # Настройка логгера
├── docs/                       # Swagger документация (генерируется)
├── tests/
│   └── service_test.go         # Unit тесты
├── go.mod
├── Dockerfile
├── config.env                  # Конфигурация окружения
└── README.md
```

## Установка

### 1. Клонирование и установка зависимостей

```bash
cd gpt/gw-currency-wallet
go mod download
```

### 2. Настройка базы данных

Создайте PostgreSQL базу данных:

```sql
CREATE DATABASE wallet_db;
CREATE USER wallet_user WITH PASSWORD 'wallet_password';
GRANT ALL PRIVILEGES ON DATABASE wallet_db TO wallet_user;
```

### 3. Настройка конфигурации

Отредактируйте `config.env`:

```env
# Server
HTTP_PORT=8080
LOG_LEVEL=info

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=wallet_user
DB_PASSWORD=wallet_password
DB_NAME=wallet_db

# JWT (ВАЖНО: измените в продакшене!)
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_EXPIRATION=24h

# Exchanger gRPC Service
EXCHANGER_GRPC_HOST=localhost
EXCHANGER_GRPC_PORT=50051

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=large-transfers
KAFKA_TRANSFER_THRESHOLD=30000
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
docker build -t gw-currency-wallet .

# Запуск контейнера
docker run -p 8080:8080 --env-file config.env gw-currency-wallet
```

## API Endpoints

### Публичные эндпоинты (без авторизации)

#### POST /api/v1/register
Регистрация нового пользователя

**Request:**
```json
{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "securepassword123"
}
```

**Response (201):**
```json
{
  "message": "User registered successfully"
}
```

#### POST /api/v1/login
Авторизация пользователя

**Request:**
```json
{
  "username": "john_doe",
  "password": "securepassword123"
}
```

**Response (200):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Защищенные эндпоинты (требуют JWT токен)

Все запросы должны содержать заголовок:
```
Authorization: Bearer <JWT_TOKEN>
```

#### GET /api/v1/balance
Получение баланса пользователя

**Response (200):**
```json
{
  "balance": {
    "USD": 1000.50,
    "EUR": 500.25,
    "RUB": 50000.00
  }
}
```

#### POST /api/v1/wallet/deposit
Пополнение счета

**Request:**
```json
{
  "amount": 100.00,
  "currency": "USD"
}
```

**Response (200):**
```json
{
  "message": "Account topped up successfully",
  "new_balance": {
    "USD": 1100.50,
    "EUR": 500.25,
    "RUB": 50000.00
  }
}
```

#### POST /api/v1/wallet/withdraw
Вывод средств

**Request:**
```json
{
  "amount": 50.00,
  "currency": "USD"
}
```

**Response (200):**
```json
{
  "message": "Withdrawal successful",
  "new_balance": {
    "USD": 1050.50,
    "EUR": 500.25,
    "RUB": 50000.00
  }
}
```

#### GET /api/v1/exchange/rates
Получение курсов валют

**Response (200):**
```json
{
  "rates": {
    "USD_EUR": 0.92,
    "USD_RUB": 92.50,
    "EUR_USD": 1.09,
    "EUR_RUB": 100.54,
    "RUB_USD": 0.0108,
    "RUB_EUR": 0.0099
  }
}
```

#### POST /api/v1/exchange
Обмен валют

**Request:**
```json
{
  "from_currency": "USD",
  "to_currency": "EUR",
  "amount": 100.00
}
```

**Response (200):**
```json
{
  "message": "Exchange successful",
  "exchanged_amount": 92.00,
  "new_balance": {
    "USD": 950.50,
    "EUR": 592.25,
    "RUB": 50000.00
  }
}
```

## Swagger документация

После запуска сервиса документация доступна по адресу:
```
http://localhost:8080/swagger/index.html
```

Для генерации Swagger документации используйте:
```bash
# Установка swag
go install github.com/swaggo/swag/cmd/swag@latest

# Генерация документации
swag init -g cmd/main.go -o docs
```

## Тестирование

### Запуск тестов

```bash
# Все тесты
go test ./... -v

# Конкретный пакет
go test ./tests -v

# С покрытием
go test ./... -cover
```

### Примеры запросов (curl)

```bash
# Регистрация
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"password123"}'

# Авторизация
TOKEN=$(curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}' \
  | jq -r '.token')

# Получение баланса
curl http://localhost:8080/api/v1/balance \
  -H "Authorization: Bearer $TOKEN"

# Пополнение
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount":1000,"currency":"USD"}'

# Обмен валют
curl -X POST http://localhost:8080/api/v1/exchange \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"from_currency":"USD","to_currency":"EUR","amount":100}'
```

## Особенности реализации

### Кеширование курсов валют

Курсы валют кешируются на 5 минут (настраивается через `CACHE_RATES_TTL`). При запросе `/api/v1/exchange`:
- Если курс запрашивался недавно (в пределах TTL) - используется кешированное значение
- Иначе выполняется gRPC запрос к exchanger сервису

### Kafka уведомления

При операциях (пополнение, вывод, обмен) с суммой более 30000 (настраивается через `KAFKA_TRANSFER_THRESHOLD`), автоматически отправляется уведомление в Kafka.

Формат сообщения:
```json
{
  "user_id": 1,
  "type": "exchange",
  "from_currency": "USD",
  "to_currency": "EUR",
  "amount": 50000.00,
  "timestamp": "2024-02-02T15:04:05Z"
}
```

### Атомарность обмена валют

Обмен валют выполняется атомарно с использованием транзакций PostgreSQL:
1. Блокировка балансов пользователя
2. Проверка достаточности средств
3. Списание исходной валюты
4. Зачисление целевой валюты
5. Создание записи о транзакции

## Безопасность

JWT токены для авторизации
Bcrypt для хеширования паролей
Валидация всех входных данных
Prepared statements против SQL injection
CORS настройки (можно добавить middleware)

## Производительность

- Connection pooling для PostgreSQL (25 открытых, 5 idle)
- Кеширование курсов валют (TTL 5 минут)
- Асинхронная отправка в Kafka
- Graceful shutdown

## Мониторинг

### Health check
```bash
curl http://localhost:8080/health
```

### Логи

Структурированное логирование в JSON формате:
```json
{
  "timestamp": "2024-02-02 15:04:05",
  "level": "info",
  "method": "POST",
  "path": "/api/v1/exchange",
  "status": 200,
  "duration": "45ms"
}
```

## Troubleshooting

### Ошибка подключения к exchanger service

Убедитесь, что gw-exchanger запущен:
```bash
curl http://localhost:50051  # или используйте grpcurl
```

### Ошибка подключения к Kafka

Kafka является опциональным компонентом. Если Kafka недоступна, сервис продолжит работу, но уведомления отправляться не будут.

### JWT токен не принимается

Проверьте:
1. Формат заголовка: `Authorization: Bearer <token>`
2. Токен не истек (24 часа по умолчанию)
3. JWT_SECRET одинаковый при генерации и проверке

## Зависимости от других сервисов

- **gw-exchanger** (обязательно) - для получения курсов валют
- **Kafka** (опционально) - для уведомлений о крупных переводах