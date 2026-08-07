# Order System

## Summary
This project is a Go-based distributed order management system responsible for the full lifecycle of creating and processing an order. It is built using an event-driven architecture across five independent microservices. Order, inventory, payment, and notification communicate exclusively through Kafka; auth-service issues the JWTs the others verify on incoming requests.

## Prerequisites
- Go 1.21+
- Docker and Docker Compose
- Air (hot reload) — `go install github.com/air-verse/air@latest`

## Getting Started
1. Clone the repository
2. Start infrastructure (PostgreSQL instances + Kafka) from the repo root:
```bash
   docker compose up -d
```
3. Run database migrations for each service:
```bash
   docker exec -i go_order_postgres psql -U postgres -d orderdb < order-service/migrations/001_init.sql
   docker exec -i go_inventory_postgres psql -U postgres -d inventorydb < inventory-service/migrations/001_init.sql
   docker exec -i go_payment_postgres psql -U postgres -d paymentdb < payment-service/migrations/001_init.sql
   docker exec -i go_auth_postgres psql -U postgres -d authdb < auth-service/migrations/001_init.sql
   docker exec -i go_notification_postgres psql -U postgres -d notificationdb < notification-service/migrations/001_init.sql
```
4. Install dependencies. `shared/` is a single Go module vendored into every service via a `replace` directive, so tidy it first:
```bash
   cd shared && go mod tidy
   cd ../order-service && go mod tidy
   cd ../inventory-service && go mod tidy
   cd ../payment-service && go mod tidy
   cd ../notification-service && go mod tidy
   cd ../auth-service && go mod tidy
```
5. Set a real `JWT_SECRET` env var for auth-service and any service that verifies tokens (order-service today). Both fall back to an insecure hardcoded default if it's unset — fine for local dev, not for anything else.
6. Start each service in a separate terminal using `air`

## Services Overview

| Service | Port | Responsibility |
|---------|------|----------------|
| auth-service | 8085 | User signup/login, JWT issuance and refresh, role-based access control |
| order-service | 8080 | Accepts incoming orders, manages order lifecycle |
| inventory-service | 8081 | Manages stock, reserves and unreserves inventory |
| payment-service | 8082 | Processes payments |
| notification-service | 8083 | Sends confirmation notifications to users |

Each service is an independent Go application with its own database, HTTP server, and (aside from auth-service) a Kafka consumer and outbox relay. They share one internal library (`shared/`) for cross-cutting concerns — logging, middleware, JWT/ULID/password helpers, validation, and Kafka event names.

## Authentication
auth-service issues short-lived JWT access tokens (15 min) plus long-lived refresh tokens (7 days), stored server-side so they can be revoked.

- `POST /auth/signup` — creates a user, hashes the password with bcrypt
- `POST /auth/login` — verifies credentials, returns an access token + refresh token
- `POST /auth/refresh` — validates the refresh token and **rotates** it: the old one is revoked and a new access/refresh pair is issued in the same transaction, so a stolen refresh token is only usable once
- `POST /auth/logout` — revokes all refresh tokens for the authenticated user

Other services verify the access token via `shared/middleware.Auth`, which decodes the JWT and attaches user ID and role to the request context; `Auth.RequireRole` gates handlers to a specific role. Both auth-service and order-service also apply a per-IP token bucket rate limiter (`shared/middleware.IPRateLimiter`) in front of their HTTP servers.

## Event Flow

Two possible paths through the order lifecycle:

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

auth-service publishes `user.registered` via its own outbox on signup, independent of the order flow above.

### Event Details

| Event | Publisher | Subscriber(s) |
|-------|-----------|---------------|
| order.created | order-service | inventory-service |
| inventory.reserved | inventory-service | payment-service |
| payment.succeeded | payment-service | notification-service, order-service |
| payment.failed | payment-service | inventory-service |
| inventory.unreserved | inventory-service | order-service |
| user.registered | auth-service | notification-service |

## Database Setup
Each service owns its own PostgreSQL database. No service accesses another service's database directly.

| Service | Database | Port |
|---------|----------|------|
| order-service | orderdb | 5432 |
| inventory-service | inventorydb | 5433 |
| payment-service | paymentdb | 5434 |
| auth-service | authdb | 5435 |
| notification-service | notificationdb | 5436 |

Every database contains at minimum:
- **outbox** — stores events to be published, ensures atomic writes with business data (notification-service is the exception: it only consumes, so it has no outbox)
- **processed_events** — tracks consumed events, ensures idempotent processing

## Reliability Patterns
- **Outbox Pattern** — events are written atomically with business data and published by a relay process, guaranteeing no lost events even if Kafka is temporarily unavailable
- **Idempotency** — every consumer checks processed_events before handling an event, preventing duplicate processing
- **Dead Letter Queue** — failed events are routed to a DLQ topic after 3 retry attempts, unblocking the consumer
- **Circuit Breaker** — payment service wraps processing with a circuit breaker, failing fast after 3 consecutive failures and recovering automatically after a cooldown period
- **Rate Limiting** — auth-service and order-service enforce a per-client-IP token bucket rate limiter (5 requests burst, 2/second sustained)
- **Refresh Token Rotation** — auth-service revokes each refresh token the moment it's used and issues a new one, limiting how long a leaked refresh token stays useful

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

A trace ID is generated at the HTTP entry point and propagated through every Kafka event, enabling full request tracing across all services.

HTTP requests are automatically logged via middleware with method, path, status code, and duration.

## Known Improvements
- **Central dependency installer** — single command to install and tidy all service modules
- **Consumer topic injection** — pass `record.Topic` into `ConsumerHandler` so handlers don't reference event names directly
- **Circuit breaker on relay** — wrap Kafka publish in the relay with a circuit breaker across all services
- **DLQ replay tool** — admin tool or automated process to replay DLQ messages after service recovery
- **Secrets management** — `JWT_SECRET` currently falls back to a hardcoded default if the env var is unset; should fail closed instead of silently using an insecure default
- **Refresh token reuse detection** — replaying an already-rotated refresh token is rejected but treated like any other invalid token; detecting reuse and revoking the whole session would flag likely token theft
- **Log level configuration** — should be configurable via environment variable per deployment environment
- **`payment.succeeded` carries no email** — notification-service's welcome/receipt email for that event falls back to `user_id` as the recipient since the event has no address on it; either the event should carry one or notification-service needs another way to resolve it
