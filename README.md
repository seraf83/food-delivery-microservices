# 🍔 Food Delivery — Microservices Demo

A food delivery backend built with microservices architecture, demonstrating async communication between services via Apache Kafka, JWT authentication via API Gateway.

## Architecture
                    ┌─────────────────┐
                    │   api-gateway   │
      Client ──────▶│   (port 8000)   │
                    │   JWT auth      │
                    └────────┬────────┘
                             │
          ┌──────────────────┼──────────────────┐
          ▼                  ▼                   ▼

┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│ user-service │ │ order-service │ │delivery-service │
│ JWT + MySQL │ │ MySQL │ │ MySQL │
└─────────────────┘ └────────┬────────┘ └────────▲────────┘
│ Kafka │
└────────────────────┘


### Services

| Service | Port | Responsibility |
|---|---|---|
| `api-gateway` | 8000 | JWT verification, request proxying |
| `user-service` | — | Registration, login, JWT issuance |
| `order-service` | — | Order creation and status tracking |
| `delivery-service` | — | Courier assignment and delivery tracking |
| `kafka-ui` | 8080 | Kafka topics browser (dev only) |

Each service has its **own MySQL database** (Database per Service pattern). Internal services are not exposed to the outside — all traffic goes through the API Gateway.

### Kafka Topics

| Topic | Producer | Consumers |
|---|---|---|
| `orders` | order-service | delivery-service |
| `deliveries` | delivery-service | order-service, user-service |

## Tech Stack

- **PHP 8.3** + **Symfony 7**
- **Doctrine ORM** — database layer
- **Symfony Messenger** — message bus abstraction
- **Apache Kafka** — async event streaming
- **Lexik JWT Authentication Bundle** — JWT auth
- **MySQL 8** — one database per service
- **Docker** + **Docker Compose**

## How It Works

1. Client registers and logs in via `api-gateway` → gets JWT token
2. Client sends `POST /api/orders` with Bearer token to `api-gateway`
3. Gateway verifies JWT, extracts `userId` and passes it via `X-User-Id` header to `order-service`
4. Order is saved to DB, `OrderPlaced` event published to Kafka
5. `delivery-service` worker consumes event → assigns courier → publishes `DeliveryAssigned`
6. `order-service` worker consumes `DeliveryAssigned` → updates order status
7. Client gets `201 Created` immediately (async, no waiting)

## Run Locally

**Requirements:** Docker, Docker Compose

```bash
git clone https://github.com/seraf83/food-delivery-microservices.git
cd food-delivery-microservices

docker compose up -d
docker compose ps

# Create Kafka topics
docker compose exec kafka kafka-topics --create --topic orders --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
docker compose exec kafka kafka-topics --create --topic deliveries --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
```

## API

**Register:**
```bash
curl -X POST http://localhost:8000/api/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@test.com", "password": "secret", "role": "customer"}'
```

**Login:**
```bash
curl -X POST http://localhost:8000/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@test.com", "password": "secret"}'
```

**Create order (with JWT):**
```bash
curl -X POST http://localhost:8000/api/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "address": "Khreschatyk 1, Kyiv",
    "items": ["Burger", "Fries"],
    "total": 360
  }'
```

**Complete delivery:**
```bash
curl -X POST http://localhost:8000/api/deliveries/{orderId}/complete \
  -H "Authorization: Bearer <token>"
```

**Browse Kafka topics:** [http://localhost:8080](http://localhost:8080)

## Key Concepts Demonstrated

- **API Gateway** — single entry point, JWT verification, request proxying
- **Microservices** — each service is independently deployable
- **Database per Service** — no shared databases between services
- **Event-Driven Architecture** — services communicate via Kafka events
- **JWT Authentication** — stateless auth, token contains userId and role
- **Async processing** — order response is immediate, courier assignment happens in background