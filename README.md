# SMS Gateway System

## English

A scalable SMS gateway service that lets customers send standard or express SMS messages, top up their wallet balance, and retrieve delivery reports. The system is designed for high throughput using Kafka, PostgreSQL, and Redis.

## Key Requirements Covered

- High volume ingestion and uneven customer distribution (Kafka + partitioning by customer).
- Express SMS path with dedicated topic and dispatcher pool.
- Strict balance enforcement (no message accepted after balance is exhausted).
- Full reporting for sent messages.

## Architecture Summary

- **API service** receives requests, validates, deducts wallet balance atomically, stores message, and publishes a Kafka event.
- **Kafka** buffers messages and allows horizontal scaling of dispatchers.
- **Dispatcher** consumes events (standard or express) and sends to the operator (simulated provider), then updates status.
- **PostgreSQL** stores customers, wallets, and message records.
- **Redis** provides distributed locks to prevent duplicate processing.

## API

All requests must include header: `X-Customer-ID`.

### Create customer

`POST /api/customers`

```json
{
  "customerId": "001",
  "name": "test name"
}
```

### Top up wallet

`POST /api/wallet/topup`

```json
{
  "amount": 1000
}
```

### Get wallet balance

`GET /api/wallet/balance`

### Send standard SMS

`POST /api/sms`

```json
{
  "to": "+989121234567",
  "text": "hello",
  "clientRef": "optional-idempotency-key"
}
```

### Send express SMS

`POST /api/sms/express`

### Get message status

`GET /api/sms/:messageId`

### List messages (report)

`GET /api/sms?status=SENT&priority=STANDARD&toNumber=+989121234567&from=2026-05-10T00:00:00Z&to=2026-05-11T00:00:00Z&limit=50&offset=0`

## Smoke Test

Run a quick end-to-end smoke test:

```bash
go test ./tests -run TestSmokeScenario
```

## Load Test

Load tests run only when explicitly enabled with `LOAD_TEST=1`.

Run:

```bash
LOAD_TEST=1 go test ./tests -run TestHighVolumeScenarios
```

Env overrides:

- `LOAD_TEST_BASE_URL` (default: http://localhost:8080)
- `LOAD_TEST_REQUESTS` (default: 500)
- `LOAD_TEST_CONCURRENCY` (default: 50)
- `LOAD_TEST_TIMEOUT_MS` (default: 10000)
- `LOAD_TEST_HIGH_TOPUP` (default: 100000)
- `LOAD_TEST_LOW_TOPUP` (default: 5)
- `LOAD_TEST_MIXED_HIGH_RATIO` (default: 0.8)

## Running

1. Start infra:

```bash
docker compose -f docker/docker-compose.yml up -d
```

2. Run migrations:

```bash
go run ./cmd/server -migration
```

3. Run API:

```bash
go run ./cmd/server
```

4. Run dispatchers:

```bash
DISPATCHER_MODE=all go run ./cmd/dispatcher
```

5. (Optional) Smoke test:

```bash
go test ./tests -run TestSmokeScenario
```

6. (Optional) Load test:

```bash
LOAD_TEST=1 go test ./tests -run TestHighVolumeScenarios
```

## Configuration

Important environment variables:

- `SERVER_PORT` (default: 8080)
- `POSTGRES_DSN` (default: postgres://sms:sms@localhost:5433/smsdb?sslmode=disable)
- `REDIS_ADDR` (default: localhost:6379)
- `KAFKA_BROKERS` (default: localhost:9092)
- `KAFKA_TOPIC_STANDARD` (default: sms-standard)
- `KAFKA_TOPIC_EXPRESS` (default: sms-express)
- `KAFKA_DLT_STANDARD` (default: sms-standard-dlt)
- `KAFKA_DLT_EXPRESS` (default: sms-express-dlt)
- `SMS_PRICE` (default: 1)
- `SMS_EXPRESS_PRICE` (default: 2)
- `EXPRESS_SLA_MS` (default: 2000)

## Notes

- Express traffic uses a dedicated Kafka topic and dispatcher to guarantee faster processing.
- Balance is deducted in a database transaction before a message is accepted.
- Redis locks avoid duplicate processing when Kafka retries occur.

## Architecture Docs

See [docs/architecture.md](docs/architecture.md).

## فارسی

این پروژه یک SMS Gateway مقیاس پذیر است که امکان ارسال پیامک استاندارد و اکسپرس، شارژ کیف پول و دریافت گزارش پیامک ها را فراهم می کند. طراحی برای حجم بالا با Kafka، PostgreSQL و Redis انجام شده است.

### نیازمندی های پوشش داده شده

- حجم بالا و توزیع نامتوازن مشتریان (Kafka با پارتیشن بندی بر اساس مشتری).
- مسیر اکسپرس با تاپیک و دیسپچر جداگانه.
- کنترل سختگیرانه موجودی (بعد از اتمام موجودی پیامک پذیرفته نمی شود).
- گزارش کامل پیامک های ارسال شده.

### API

تمام درخواست ها باید هدر `X-Customer-ID` داشته باشند.

ایجاد مشتری

`POST /api/customers`

```json
{
  "customerId": "001",
  "name": "test name"
}
```

شارژ کیف پول

`POST /api/wallet/topup`

```json
{
  "amount": 1000
}
```

مشاهده موجودی

`GET /api/wallet/balance`

ارسال پیامک استاندارد

`POST /api/sms`

ارسال پیامک اکسپرس

`POST /api/sms/express`

مشاهده وضعیت پیام

`GET /api/sms/:messageId`

گزارش پیامک ها

`GET /api/sms?status=SENT&priority=STANDARD&toNumber=+989121234567&from=2026-05-10T00:00:00Z&to=2026-05-11T00:00:00Z&limit=50&offset=0`

### اجرا

1. اجرای زیرساخت:

```bash
docker compose -f docker/docker-compose.yml up -d
```

2. اجرای مایگریشن:

```bash
go run ./cmd/server -migration
```

3. اجرای API:

```bash
go run ./cmd/server
```

4. اجرای دیسپچرها:

```bash
DISPATCHER_MODE=all go run ./cmd/dispatcher
```

5. (اختیاری) تست سریع:

```bash
go test ./tests -run TestSmokeScenario
```

6. (اختیاری) تست بار:

```bash
LOAD_TEST=1 go test ./tests -run TestHighVolumeScenarios
```

### تست بار

تست بار فقط با فعال سازی صریح `LOAD_TEST=1` اجرا می شود.

```bash
LOAD_TEST=1 go test ./tests -run TestHighVolumeScenarios
```

متغیرهای تنظیم:

- `LOAD_TEST_BASE_URL` (پیش فرض: http://localhost:8080)
- `LOAD_TEST_REQUESTS` (پیش فرض: 500)
- `LOAD_TEST_CONCURRENCY` (پیش فرض: 50)
- `LOAD_TEST_TIMEOUT_MS` (پیش فرض: 10000)
- `LOAD_TEST_HIGH_TOPUP` (پیش فرض: 100000)
- `LOAD_TEST_LOW_TOPUP` (پیش فرض: 5)
- `LOAD_TEST_MIXED_HIGH_RATIO` (پیش فرض: 0.8)
