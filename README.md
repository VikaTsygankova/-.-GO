# Bank Service REST API

Учебный REST API банковского сервиса на Go.

Проект реализует требования итогового задания: регистрация, JWT-аутентификация, счета, карты, переводы, кредиты, график платежей, аналитика, PostgreSQL, SMTP-заготовка, интеграция с ЦБ РФ через SOAP, bcrypt, HMAC и pgcrypto/PGP-заготовка.

## Технологии

- Go 1.23+
- gorilla/mux
- PostgreSQL 17
- lib/pq
- JWT: github.com/golang-jwt/jwt/v5
- bcrypt
- HMAC-SHA256
- logrus
- gomail.v2
- beevik/etree

## Быстрый запуск

### 1. Создать базу

```sql
CREATE DATABASE bank_service;
```

### 2. Настроить переменные окружения


Минимально нужно:

```text
DATABASE_URL=postgres://postgres:postgres@localhost:5432/bank_service?sslmode=disable
JWT_SECRET=change-me-super-secret-key
HMAC_SECRET=change-me-hmac-secret
SERVER_PORT=8080
```

### 3. Установить зависимости

```bash
go mod tidy
```

### 4. Запустить

```bash
go run ./cmd/server
```

После запуска API доступно по адресу:

```text
http://localhost:8080
```

При первом запуске приложение само создаёт таблицы.

## API

### Регистрация

```http
POST http://localhost:8080/register
```

```json
{
  "username": "vika",
  "email": "vika@example.com",
  "password": "12345678"
}
```

### Логин

```http
POST http://localhost:8080/login
```

```json
{
  "email": "vika@example.com",
  "password": "12345678"
}
```

Ответ содержит JWT token.

Для защищённых запросов:

```http
Authorization: Bearer YOUR_TOKEN
```

## Защищённые endpoints

### Создать счёт

```http
POST http://localhost:8080/accounts
```

```json
{
  "name": "Main account"
}
```

### Получить свои счета

```http
GET http://localhost:8080/accounts
```

### Пополнить счёт

```http
POST http://localhost:8080/accounts/1/deposit
```

```json
{
  "amount": 10000
}
```

### Списать со счёта

```http
POST http://localhost:8080/accounts/1/withdraw
```

```json
{
  "amount": 500
}
```

### Перевод между счетами

```http
POST http://localhost:8080/transfer
```

```json
{
  "from_account_id": 1,
  "to_account_id": 2,
  "amount": 1000
}
```

### Выпустить карту

```http
POST http://localhost:8080/cards
```

```json
{
  "account_id": 1
}
```

Карта генерируется с валидным номером по алгоритму Луна. Номер и срок сохраняются в зашифрованном виде, CVV хранится как bcrypt hash, HMAC используется для проверки целостности.

### Получить свои карты

```http
GET http://localhost:8080/cards
```

### Оплата картой

```http
POST http://localhost:8080/cards/pay
```

```json
{
  "card_id": 1,
  "amount": 300,
  "merchant": "Test Shop"
}
```

### Оформить кредит

```http
POST http://localhost:8080/credits
```

```json
{
  "account_id": 1,
  "amount": 50000,
  "months": 12
}
```

Процентная ставка берётся через сервис ЦБ РФ + маржа банка. Если ЦБ недоступен, используется резервная ставка.

### График платежей

```http
GET http://localhost:8080/credits/1/schedule
```

### Аналитика

```http
GET http://localhost:8080/analytics
```

### Прогноз баланса

```http
GET http://localhost:8080/accounts/1/predict?days=30
```

Максимум — 365 дней.


## Структура проекта

```text
cmd/server/main.go
internal/config
internal/db
internal/models
internal/repository
internal/service
internal/handler
internal/middleware
internal/utils
```

## Примечания

- Валюта: только RUB.
- JWT живёт 24 часа.
- Scheduler обработки кредитных платежей запускается каждые 12 часов.
- SMTP параметры настраиваются через переменные окружения.
- Для учебного запуска PGP реализован через pgcrypto-compatible placeholder: данные карт не хранятся в открытом виде, а дополнительно защищаются HMAC.
