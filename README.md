# MedStock - Inventory Sync Engine

**Live Demo:** https://med-stock-three.vercel.app/

A production-grade B2B inventory synchronization engine for medical device suppliers, built with Go, PostgreSQL, Redis, and Next.js.

## Architecture

```
medstock/
├── cmd/api/           # Go API entry point
├── internal/
│   ├── domain/        # Domain entities (Product, Inventory, Upload)
│   ├── application/   # Business logic services
│   ├── infrastructure/# PostgreSQL, Redis, HTTP handlers
│   ├── worker/        # Worker pool with backpressure
│   └── middleware/     # Auth, CORS, logging
├── pkg/               # Shared utilities (logger, validator)
├── migrations/        # PostgreSQL schema
├── frontend/          # Next.js 16 + React Query
└── scripts/          # Utility scripts
```

## Features

- **High-throughput bulk uploads** (up to 10,000 items per request)
- **ACID-compliant transactions** with optimistic locking
- **Worker pool with semaphore-based backpressure**
- **Redis caching** for hot data
- **Real-time progress tracking** with WebSocket-ready architecture
- **Idempotent uploads** via idempotency keys
- **Chunked processing** for large inventories

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.21+ |
| API | Gorilla Mux |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| Frontend | Next.js 16 |
| State | React Query |
| Styling | Tailwind CSS |

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL 16
- Redis 7

### Backend (API Server)

```bash
# Clone and navigate
cd medstock

# Run with mock database (no external deps needed)
go run ./cmd/api/main.go

# Or with Docker Compose for full stack
docker-compose up -d
go run ./cmd/api/main.go
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:3000

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/healthz` | Health check |
| POST | `/api/v1/bulk-upload` | Bulk inventory upload |
| GET | `/api/v1/products` | List all products |
| GET | `/api/v1/uploads/{id}` | Get upload status |
| GET | `/api/v1/stats` | Get inventory stats |

## Example: Bulk Upload

```bash
curl -X POST http://localhost:8080/api/v1/bulk-upload \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "idempotency_key": "upload-123",
    "items": [
      {"sku": "MED-001", "name": "Surgical Mask", "quantity": 1000, "unit_price": 0.50}
    ]
  }'
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | API server port |
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `REDIS_ADDR` | localhost:6379 | Redis address |

## Interview Highlights

This project demonstrates:

1. **Concurrency** - Bounded worker pool with semaphore-based backpressure
2. **Consistency** - ACID transactions, optimistic locking, idempotency keys
3. **Caching** - Cache-aside pattern with invalidation on writes
4. **Observability** - Structured logging with zerolog
5. **Architecture** - Clean separation (domain, application, infrastructure)

## License

MIT