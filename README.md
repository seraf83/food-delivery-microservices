# 🍔 Food Delivery — Microservices Demo

A food delivery backend built with microservices architecture, demonstrating async communication between services via Apache Kafka.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  order-service  │────▶│     Kafka        │────▶│delivery-service │
│   (port 8002)   │     │  (port 9092)     │     │   (port 8003)   │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                  │
                                  ▼
                        ┌─────────────────┐
                        │  user-service   │
                        │   (port 8001)   │
                        └─────────────────┘
```

### Services

| Service | Port | Responsibility |
|---|---|---|
| `user-service` | 8001 | User management, notifications |
| `order-service` | 8002 | Order creation and status tracking |
| `delivery-service` | 8003 | Courier assignment and delivery tracking |
| `kafka-ui` | 8080 | Kafka topics browser (dev only) |

Each service has its **own MySQL database** (Database per Service pattern).

### Kafka Topics

| Topic | Producer | Consumers |
|---|---|---|
| `order.created` | order-service | delivery-service, user-service |
| `delivery.assigned` | delivery-service | order-service, user-service |

## Tech Stack

- **PHP 8.4** + **Symfony 6**
- **Doctrine ORM** — database layer
- **Symfony Messenger** — message bus abstraction
- **Apache Kafka** — async event streaming
- **MySQL 8** — one database per service
- **Docker** + **Docker Compose**

## How It Works

1. Client sends `POST /orders` to `order-service`
2. Order is saved to DB
3. `order-service` publishes `OrderPlaced` event to Kafka
4. `delivery-service` worker consumes the event → assigns a courier → publishes `DeliveryAssigned`
5. `user-service` worker consumes the event → sends SMS notification to the user
6. Client gets `201 Created` response immediately (async, no waiting)

## Run Locally

**Requirements:** Docker, Docker Compose

```bash
# Clone the repo
git clone https://github.com/YOUR_USERNAME/food-delivery-microservices.git
cd food-delivery-microservices

# Start all services
docker compose up -d

# Check all containers are running
docker compose ps
```

## Test the Flow

**Create an order:**
```bash
curl -X POST http://localhost:8002/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 1,
    "address": "Khreschatyk 1, Kyiv",
    "items": [
      {"name": "Burger", "qty": 2, "price": 150},
      {"name": "Fries",  "qty": 1, "price": 60}
    ],
    "total": 360
  }'
```

**Watch workers process the event:**
```bash
docker compose logs -f order-worker delivery-worker user-worker
```

You should see:
```
order-worker      | ✅ Order #1 saved
delivery-worker   | 🛵 Courier Богдан Коваль assigned to order #1
user-worker       | 📱 SMS → order#1: Courier is on the way! ETA: 41 min
```

**Browse Kafka topics:**

Open [http://localhost:8080](http://localhost:8080) in your browser.

## Project Structure

```
food-delivery/
├── docker-compose.yml
├── order-service/
│   ├── src/
│   │   ├── Controller/OrderController.php
│   │   ├── Entity/Order.php
│   │   ├── Message/OrderPlaced.php
│   │   └── MessageHandler/DeliveryAssignedHandler.php
│   └── Dockerfile
├── delivery-service/
│   ├── src/
│   │   ├── MessageHandler/OrderPlacedHandler.php
│   │   └── Message/DeliveryAssigned.php
│   └── Dockerfile
└── user-service/
    ├── src/
    │   └── MessageHandler/DeliveryAssignedHandler.php
    └── Dockerfile
```

## Key Concepts Demonstrated

- **Microservices** — each service is independently deployable
- **Database per Service** — no shared databases between services
- **Event-Driven Architecture** — services communicate via Kafka events, not direct HTTP calls
- **Async processing** — order response is immediate, courier assignment happens in background
- **Message Bus pattern** — Symfony Messenger abstracts the transport (swap Kafka for RabbitMQ/Redis with config change only)
