# Order System

## Summary
This project is a Go-based distributed order management system responsible for the full lifecycle of creating and processing an order. It is built using an event-driven architecture across four independent microservices that communicate exclusively through Kafka.

## Prerequisites
- Go 1.21+
- Docker and Docker Compose
- Air (hot reload) — `go install github.com/air-verse/air@latest`

## Getting Started
1. Clone the repository
2. Start infrastructure (PostgreSQL instances + Kafka):
```bash
   cd order-service
   docker compose up -d
```
3. Run database migrations for each service:
```bash
   docker exec -i go_order_postgres psql -U postgres -d orderdb < order-service/storage/migrations.sql
   docker exec -i go_inventory_postgres psql -U postgres -d inventorydb < inventory-service/storage/migrations.sql
   docker exec -i go_payment_postgres psql -U postgres -d paymentdb < payment-service/storage/migrations.sql
```
4. Install dependencies in each service:
```bash
   cd order-service && go mod tidy
   cd ../inventory-service && go mod tidy
   cd ../payment-service && go mod tidy
   cd ../notification-service && go mod tidy
   cd ../shared/logger && go mod tidy
   cd ../events && go mod tidy
```
5. Start each service in a separate terminal using `air`

## Services Overview

| Service | Port | Responsibility |
|---------|------|----------------|
| order-service | 8080 | Accepts incoming orders, manages order lifecycle |
| inventory-service | 8081 | Manages stock, reserves and unreserves inventory |
| payment-service | 8082 | Processes payments |
| notification-service | 8083 | Sends confirmation notifications to users |

Each service is an independent Go module with its own database, Kafka consumer, outbox relay, and HTTP server.

## Event Flow

Two possible paths through the system:

**Happy path (payment succeeds):**
POST /orders
- order.created
- inventory.reserved
- payment.succeeded
- notification sent + order confirmed

**Compensation path (payment fails):**
POST /orders
- order.created
- inventory.reserved
- payment.failed
- inventory.unreserved
- order marked as failed

### Event Details

| Event | Publisher | Subscriber(s) |
|-------|-----------|---------------|
| order.created | order-service | inventory-service |
| inventory.reserved | inventory-service | payment-service |
| payment.succeeded | payment-service | notification-service, order-service |
| payment.failed | payment-service | inventory-service |
| inventory.unreserved | inventory-service | order-service |

## Database Setup
Each service owns its own PostgreSQL database. No service accesses another service's database directly.

| Service | Database | Port |
|---------|----------|------|
| order-service | orderdb | 5432 |
| inventory-service | inventorydb | 5433 |
| payment-service | paymentdb | 5434 |

Every database contains at minimum:
- **outbox** — stores events to be published, ensures atomic writes with business data
- **processed_events** — tracks consumed events, ensures idempotent processing

## Reliability Patterns
- **Outbox Pattern** — events are written atomically with business data and published by a relay process, guaranteeing no lost events even if Kafka is temporarily unavailable
- **Idempotency** — every consumer checks processed_events before handling an event, preventing duplicate processing
- **Dead Letter Queue** — failed events are routed to a DLQ topic after 3 retry attempts, unblocking the consumer
- **Circuit Breaker** — payment service wraps processing with a circuit breaker, failing fast after 3 consecutive failures and recovering automatically after a cooldown period
- **Rate Limiting** — order service enforces a token bucket rate limiter (5 requests burst, 2/second sustained)

## Logging
All services use a shared structured logger (`shared/logger`) built on zerolog. Every log line is JSON with consistent fields:

```json
{
  "level": "info",
  "trace_id": "1784591313965949000",
  "service": "order-service",
  "eventName": "order.created",
  "time": "2026-07-21T00:48:33+01:00",
  "message": "Order confirmed"
}
```

A trace ID is generated at the HTTP entry point and propagated through every Kafka event, enabling full request tracing across all four services.

HTTP requests are automatically logged via middleware with method, path, status code, and duration.

## Known Improvements
- **Central dependency installer** — single command to install and tidy all service modules
- **Consumer topic injection** — pass `record.Topic` into `ConsumerHandler` so handlers don't reference event names directly
- **Circuit breaker on relay** — wrap Kafka publish in the relay with a circuit breaker across all services
- **DLQ replay tool** — admin tool or automated process to replay DLQ messages after service recovery
- **Environment variables** — database URLs, Kafka brokers, ports, and log levels are currently hardcoded
- **Docker Compose at root level** — currently lives inside order-service, should be at system root
- **Log level configuration** — should be configurable via environment variable per deployment environment