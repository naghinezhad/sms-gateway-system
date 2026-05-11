# SMS Gateway System - Architecture

## Goals

- Handle very high throughput (up to 100M SMS/day).
- Support uneven customer traffic distribution.
- Provide express delivery path with a defined SLA.
- Enforce strict wallet balance rules.
- Offer detailed reporting for customers.

## Context Diagram

```mermaid
flowchart LR
  Customer[Customer App] --> API[SMS Gateway API]
  API --> Kafka[(Kafka Topics)]
  Kafka --> Dispatcher[Dispatcher Workers]
  Dispatcher --> Operator[SMS Operator]
  API --> Postgres[(PostgreSQL)]
  API --> Redis[(Redis)]
  Dispatcher --> Postgres
  Dispatcher --> Redis
```

## Container Diagram

```mermaid
flowchart TB
  subgraph SMS_Gateway_System
    API[API Service
Gin + Go]
    DispatcherStd[Dispatcher Standard]
    DispatcherExp[Dispatcher Express]
  end

  API --> Postgres[(PostgreSQL)]
  API --> Redis[(Redis Locking)]
  API --> KafkaStd[(Kafka: sms-standard)]
  API --> KafkaExp[(Kafka: sms-express)]

  DispatcherStd --> KafkaStd
  DispatcherExp --> KafkaExp
  DispatcherStd --> Postgres
  DispatcherExp --> Postgres
  DispatcherStd --> Redis
  DispatcherExp --> Redis
  DispatcherStd --> Operator[SMS Operator]
  DispatcherExp --> Operator
```

## Sequence Diagram - Send SMS

```mermaid
sequenceDiagram
  participant C as Customer
  participant API as API Service
  participant DB as PostgreSQL
  participant K as Kafka
  participant D as Dispatcher
  participant O as Operator

  C->>API: POST /api/sms (X-Customer-ID)
  API->>DB: Begin TX
  API->>DB: Lock wallet row (FOR UPDATE)
  API->>DB: Check balance + create message
  API->>DB: Deduct balance
  API->>DB: Commit TX
  API->>K: Publish sms-send event
  API-->>C: 202 Accepted (messageId)

  D->>K: Consume event
  D->>O: Send SMS
  D->>DB: Update status (SENT/FAILED)
```

## Key Design Decisions

- **Wallet enforcement**: balance check and deduction are done in a single database transaction with row-level locking to prevent overspending.
- **Express path**: express messages go to a dedicated Kafka topic and dispatcher pool to reduce queueing delay.
- **Partitioning**: Kafka message key is `customer_id` to preserve per-customer ordering while allowing horizontal scaling.
- **Redis locks**: prevent duplicate processing when events are retried or re-delivered.
- **SLA**: express SLA is measured from message acceptance time to operator send time. Breaches are flagged.

## Scaling Notes

- Increase Kafka partitions and dispatcher replicas to handle higher throughput.
- Use separate dispatcher deployments for express and standard.
- Add a dedicated outbox relay if exactly-once publication is required.
