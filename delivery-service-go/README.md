# delivery-service (Go)

[![delivery-service-go](https://github.com/seraf83/food-delivery-microservices/actions/workflows/delivery-service-go.yml/badge.svg)](https://github.com/seraf83/food-delivery-microservices/actions/workflows/delivery-service-go.yml)

Assigns couriers to new orders and tracks deliveries. A Go rewrite of the former
Symfony service (kept in [`../delivery-service-legacy`](../delivery-service-legacy)).
It is a drop-in replacement: same HTTP API, same Kafka message format, same
database schema.

## How it works

One binary, two modes:

| Mode | Command | What it does |
|------|---------|--------------|
| HTTP API | `delivery-service serve` | REST endpoints for delivery status and completion |
| Worker | `delivery-service consume` | Reads `OrderPlaced` from Kafka, assigns a courier, publishes `DeliveryAssigned` |

```
order-service ──OrderPlaced──▶ [orders] ──▶ worker ──▶ MySQL (assign courier)
                                               │
                                               └──DeliveryAssigned──▶ [deliveries] ──▶ order-service, user-service

api-gateway ──POST /deliveries/{id}/complete──▶ HTTP API ──▶ MySQL (free courier)
                                                   │
                                                   └──DeliveryCompleted──▶ [deliveries]
```

## Project layout

```
cmd/delivery-service/   entry point: config, wiring, graceful shutdown
internal/config/        settings from environment variables
internal/store/         MySQL layer (database/sql + go-sql-driver/mysql)
internal/events/        Kafka message contract and publisher (segmentio/kafka-go)
internal/httpapi/       HTTP handlers (net/http ServeMux, Go 1.22+ routing patterns)
internal/worker/        Kafka consumer loop
```

## HTTP API

Not exposed directly — clients go through `api-gateway`, which verifies the JWT
and forwards `X-User-Id` / `X-User-Role` headers.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/deliveries/{orderId}` | Delivery status. Customers only see their own deliveries (others get `404`) |
| `POST` | `/deliveries/{orderId}/complete` | Mark delivered, free the courier, publish `DeliveryCompleted` |
| `GET` | `/couriers` | List couriers and their status |
| `GET` | `/health` | Liveness check |

## Kafka messages

Messages are JSON envelopes shared with the PHP services:

```json
{"type": "App\\Message\\DeliveryAssigned", "data": {"orderId": 1, "courierId": 2, ...}}
```

The PHP consumers build the message object with
`new $class(...array_values($data))`, so **field order in the Go structs must
match the PHP constructor**. `events_test.go` guards this.

| Event | Topic | Direction |
|-------|-------|-----------|
| `OrderPlaced` | `orders` | consumed |
| `DeliveryAssigned` | `deliveries` | produced |
| `DeliveryCompleted` | `deliveries` | produced |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — (required) | `mysql://user:pass@host:port/db`, same format as the PHP services |
| `HTTP_ADDR` | `:80` | HTTP listen address |
| `KAFKA_DSN` | `kafka:9092` | Comma-separated broker list |
| `KAFKA_ORDERS_TOPIC` | `orders` | Topic to consume |
| `KAFKA_DELIVERIES_TOPIC` | `deliveries` | Topic to publish to |
| `KAFKA_GROUP_ID` | `delivery-consumer` | Consumer group |

Tables are created on startup if missing (`CREATE TABLE IF NOT EXISTS`) with the
schema of the old Doctrine migration, so an existing database works unchanged.

## Running

With the whole stack, from the repository root:

```bash
docker compose up -d --build delivery-service delivery-worker
docker compose logs -f delivery-worker
```

Locally (MySQL and Kafka must be reachable):

```bash
export DATABASE_URL=mysql://delivery_user:delivery_pass@localhost:3308/delivery
export KAFKA_DSN=localhost:9092 HTTP_ADDR=:8003
go run ./cmd/delivery-service serve
go run ./cmd/delivery-service consume
```

## Development

```bash
go vet ./...
go test -race ./...         # needs Docker: store tests start MySQL via testcontainers
go test -short ./...        # unit tests only, no Docker
golangci-lint run
```

| Package | Tests |
|---------|-------|
| `config` | `DATABASE_URL` conversion, env defaults |
| `events` | JSON field order matches the PHP constructors |
| `httpapi` | customers only see their own deliveries |
| `worker` | message handling with fake store/publisher; offsets are committed only after success |
| `store` | integration tests on real MySQL 8: courier assignment, redelivery, completion, and concurrent assignment (fails without `SKIP LOCKED`) |

CI runs the same checks on every push and pull request that touches this
directory (`.github/workflows/delivery-service-go.yml`).

## Design notes

- **At-least-once processing.** The worker commits a Kafka offset only after the
  message is fully handled. On a transient error (DB or Kafka down) it exits
  without committing; Docker restarts it and the message is read again.
- **Idempotency.** Redelivered `OrderPlaced` events are detected by an existing
  delivery for the order and are not published again, so a stale
  `DeliveryAssigned` can't overwrite a later order status. Completing a delivery
  twice returns `200` but frees no courier and publishes no second event.
- **Concurrent workers.** Courier selection uses
  `SELECT ... FOR UPDATE SKIP LOCKED`, so several workers in the same consumer
  group never assign the same courier.
- **Graceful shutdown.** `SIGTERM` cancels the root context; the HTTP server
  drains requests and the consumer stops between messages.
- **Small image.** Multi-stage build into `distroless/static:nonroot`.

### Known limitations

- **No transactional outbox.** The DB commit and the Kafka publish are separate
  steps. If the process dies between them, the courier is assigned but
  `DeliveryAssigned` is never sent (the redelivered event is treated as a
  duplicate). An outbox table plus a relay would fix this.
- No unique index on `deliveries.order_id` (kept for schema compatibility with
  the legacy service).
- Poison messages with a valid envelope but a failing handler block the
  partition; there is no dead-letter topic yet.
