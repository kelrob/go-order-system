# ADR-001: Microservices Architecture

## Status: Accepted

## Context
The order system needed to handle multiple distinct responsibilities — order creation, inventory management, payment processing, and notifications. The choice was between building a single monolithic application or splitting into independent services. The system needed to be resilient — a failure in payment processing should not take down order creation.

## Decision
We chose a microservices architecture with four independent services — order, inventory, payment, and notification. Each service owns its own database, runs its own process, and communicates exclusively through Kafka events. No service calls another directly.

## Consequences
**Gains:**
- Fault isolation — a payment service failure does not affect order creation
- Independent deployability — each service can be deployed and scaled independently
- Clear ownership — each service has a single responsibility

**Tradeoffs:**
- Operational complexity — four services to run, monitor, and maintain instead of one
- Distributed systems challenges — network failures, eventual consistency, and distributed tracing required
- More infrastructure — separate databases, Kafka, and health checks per service

# ADR-002: Kafka for Inter-Service Communication

## Status: Accepted

## Context
With four independent microservices, we needed a way for them to communicate without direct coupling. Options considered:
- **Direct REST calls** — simple but creates tight coupling. If the inventory service is down, the order service fails too
- **RabbitMQ** — good message queue but deletes messages after consumption, limiting replay capability
- **Kafka** — append-only distributed log, messages retained after consumption, multiple consumers can read the same event independently

## Decision
We chose Kafka as the communication layer between all four services. Services publish events to Kafka topics and subscribe to the events they care about. No service calls another directly.

## Consequences
**Gains:**
- Loose coupling — services don't know about each other, only about events
- Message retention — events are stored in Kafka and can be replayed if a service was down
- Multiple consumers — one event can be consumed by multiple services independently
- Async processing — order service returns to the customer immediately without waiting for downstream services

**Tradeoffs:**
- Operational overhead — Kafka requires Zookeeper, topic management, and monitoring
- Eventual consistency — downstream services process events asynchronously, state is not immediately consistent across all services
- Local setup complexity — requires Docker and proper configuration to run locally


# ADR-003: Outbox Pattern for Reliable Event Publishing

## Status: Accepted

## Context
After choosing Kafka for inter-service communication, we faced the dual-write problem. Every time a business operation occurs (e.g. an order is created), we need to both write to the database AND publish a Kafka event. These are two separate systems with no shared transaction boundary. If the database write succeeds but Kafka publish fails, the event is lost and downstream services never react. If Kafka publish succeeds but the database write fails, we have a phantom event with no corresponding data.

## Decision
We adopted the outbox pattern. Instead of writing to the database and Kafka separately, every service writes to its own database in a single transaction — inserting the business data and an outbox entry atomically. A relay process polls the outbox table every 2 seconds, publishes pending entries to Kafka, and marks them as published. If Kafka is unavailable, entries remain pending until the relay successfully publishes them.

## Consequences
**Gains:**
- Atomicity — business data and event are always written together or not at all
- No lost events — if Kafka is temporarily down, events queue in the outbox and are published when it recovers
- Auditability — the outbox table provides a history of all published events

**Tradeoffs:**
- Additional complexity — each service requires an outbox table, a relay process, and a processed_events table for idempotency
- Relay is a dependency — if the relay process is down, events are delayed until it recovers. The outbox entries are safe but not yet published
- Polling delay — the relay polls every 2 seconds, introducing up to 2 seconds of latency between a business operation and its downstream effects


# ADR-004: Choreography-Based Saga for Distributed Transactions

## Status: Accepted

## Context
With four independent services each owning their own database, we needed a way to coordinate a multi-step business process — create order, reserve inventory, process payment, send notification — where any step could fail and require compensation. Two options were considered:
- **Orchestration** — a central coordinator service that calls each service directly and manages the full flow and compensation logic
- **Choreography** — each service reacts to events independently, triggering the next step by publishing its own event

## Decision
We chose the choreography pattern. Each service subscribes only to the events it cares about and publishes events when its work is complete. No central coordinator exists. The flow emerges naturally from the events:
- Inventory service reacts to order.created
- Payment service reacts to inventory.reserved
- Notification and order services react to payment.succeeded
- Inventory service reacts to payment.failed and compensates automatically

## Consequences
**Gains:**
- Loose coupling — services don't know about each other, only about events
- No single point of failure — there is no orchestrator whose downtime halts the entire flow
- Easy to extend — adding a new service only requires subscribing to an existing event, no other service needs to change
- Each service is fully autonomous

**Tradeoffs:**
- Distributed visibility — there is no single place to see the full flow. Tracing a failed order requires searching logs across multiple services. Mitigated by structured logging with a consistent trace ID propagated through all events
- Implicit flow — the business process is not defined in one place, it emerges from event subscriptions across services. New engineers must read multiple services to understand the full flow
- Not ideal for complex flows — if the business process grows to 15+ steps with many branches, an orchestrator would provide better control and visibility

# ADR-005: Database Per Service

## Status: Accepted

## Context
With four independent microservices, we needed to decide on data ownership. Two options:
- **Shared database** — all services read and write to one central database. Simple to query across domains but creates tight coupling at the data layer
- **Database per service** — each service owns its own database. No other service reads or writes to it directly

## Decision
We chose database per service. Each service is the single source of truth for its own data. The order service owns orders, the inventory service owns stock levels, the payment service owns payment records. Services never access each other's databases directly — they communicate only through Kafka events.

## Consequences
**Gains:**
- True independence — each service can be deployed, scaled, and maintained without affecting others
- Fault isolation — if the payment database goes down, order creation is unaffected
- Freedom to optimise — each service can choose the database type and schema that best fits its needs
- Clear data ownership — no ambiguity about which service is responsible for which data

**Tradeoffs:**
- No cross-service joins — querying data that spans multiple services (e.g. order details with payment status) requires calling each service separately and merging in the application layer
- Multiple databases to manage — more infrastructure, more cost, more operational overhead
- Data duplication — some fields like user_id and order_id appear in multiple databases by design
- Eventual consistency — data across services is consistent only after events have been processed, not immediately


# ADR-006: PostgreSQL as the Database Engine

## Status: Accepted

## Context
Each service requires its own database. We needed to choose a database engine that fits our data model and reliability requirements. Options considered:
- **NoSQL (MongoDB, DynamoDB)** — flexible schema, easy to scale horizontally, but weak transaction support
- **MySQL** — solid relational database but less feature-rich than PostgreSQL
- **SQLite** — simple but not suitable for production concurrent workloads
- **PostgreSQL** — full-featured relational database with strong ACID guarantees and advanced data types

## Decision
We chose PostgreSQL for all four service databases. The decision was driven by:
- **Relational data** — orders have order items, requiring a one-to-many relationship across two tables with JOIN queries
- **ACID guarantees** — the outbox pattern depends entirely on atomic transactions. Writing business data and outbox entries in one transaction requires a database that guarantees atomicity, consistency, isolation and durability
- **JSONB support** — outbox event payloads are stored as JSONB, allowing structured JSON storage with efficient querying without a separate document store
- **Maturity and reliability** — PostgreSQL is battle-tested for production workloads

## Consequences
**Gains:**
- ACID transactions — critical for the outbox pattern and data integrity across multi-step operations
- Relational queries — JOIN across orders and order_items, foreign key constraints, complex filtering
- JSONB — flexible event payload storage without sacrificing query capability
- Strong ecosystem — excellent Go drivers (pgx), tooling, and community support

**Tradeoffs:**
- Operational overhead — each service has its own PostgreSQL instance to manage, monitor, and back up
- Vertical scaling — PostgreSQL scales vertically more naturally than horizontally. At very high write volumes a NoSQL database may be more appropriate for certain services
- Schema migrations — as the system evolves, schema changes require careful migration management across four separate databases