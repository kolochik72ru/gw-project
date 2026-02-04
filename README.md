# ИНСТРУКЦИЯ ПО ЗАПУСКУ

### Cтруктура:

```
gw-project/
├── gw-exchanger/
│   ├── proto/
│   │   └── exchange/
│   │       └── exchange.proto
│   ├── generate-proto.sh       
│   ├── cmd/
│   ├── internal/
│   └── go.mod 
│
├── gw-currency-wallet/
│   ├── proto/
│   │   └── exchange/
│   │       └── exchange.proto
│   ├── generate-proto.sh       
│   ├── cmd/
│   ├── internal/
│   └── go.mod 
│
├── gw-notification/
│   ├── cmd/
│   ├── internal/
│   └── go.mod (go 1.24)
│
└── docker-compose.yml
```

## ЗАПУСК 

```bash
docker-compose up -d
```

**Что запускается:**
- PostgreSQL для exchanger (порт 5432)
- PostgreSQL для wallet (порт 5433)
- MongoDB (порт 27017)
- Zookeeper + Kafka (порты 2181, 9092)
- gw-exchanger (порт 50051)
- gw-currency-wallet (порт 8080)
- gw-notification

### Проверить что работает

```bash
# Проверка статуса
docker-compose ps

# Health check
curl http://localhost:8080/health
# Ответ: {"status":"ok"}
```

## БЫСТРЫЙ ТЕСТ СИСТЕМЫ

### 1. Регистрация пользователя

```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test_user",
    "email": "test@example.com",
    "password": "password123"
  }'
```

Ответ: `{"message":"User registered successfully"}`

### 2. Авторизация

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test_user",
    "password": "password123"
  }' | jq -r '.token')

echo "Token: $TOKEN"
```

### 3. Пополнение на 100,000 USD (создаст уведомление!)

```bash
curl -X POST http://localhost:8080/api/v1/wallet/deposit \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 100000,
    "currency": "USD"
  }'
```

Ответ: `{"message":"Account topped up successfully","new_balance":{...}}`

### 4. Проверить уведомление в MongoDB

```bash
docker exec -it mongodb mongosh

use notification_db
db.large_transfers.find().pretty()
```

Увидишь запись о пополнении 100,000 USD!

## КАК ОСТАНОВИТЬ

```bash
# Остановить всё
docker-compose down

# Остановить и удалить данные
docker-compose down -v
```
