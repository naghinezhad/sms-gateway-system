# SMS Gateway System

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

`GET /api/sms?status=SENT&priority=STANDARD&from=2026-05-10T00:00:00Z&to=2026-05-11T00:00:00Z&limit=50&offset=0`

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
DISPATCHER_MODE=standard go run ./cmd/dispatcher
DISPATCHER_MODE=express go run ./cmd/dispatcher
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
