# Job Platform — Distributed Job Processing System

A distributed job processing platform built with **Go**, **React**, **PostgreSQL**, **RabbitMQ**, and **Redis**. Submit long-running tasks via REST API, process them asynchronously with a fleet of workers, and monitor everything through a real-time dashboard.

## Architecture


```
┌──────────────┐       ┌──────────────┐       ┌───────────────┐
│   React UI   │──────▶│   Go API     │──────▶│  PostgreSQL   │
│  (Vite+TS)   │       │  (Chi)       │──TX──▶│  + Outbox     │
└──────────────┘       └──────┬───────┘       └───────────────┘
                              │                       ▲
                              │ Publish               │ Poll
                              ▼                       │
                       ┌──────────────┐       ┌───────┴───────┐
                       │   RabbitMQ   │◀──────│ Outbox        │
                       │              │       │ Publisher      │
                       └──────┬───────┘       └───────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
             ┌────────────┐     ┌────────────┐
             │  Worker 1  │     │  Worker N  │
             └────────────┘     └────────────┘
```

## Key Features

| Category | Feature |
|----------|---------|
| **API** | RESTful Go API with Chi router, JWT auth, request IDs |
| **Reliability** | Transactional outbox pattern, idempotency keys, retry with exponential backoff |
| **State Machine** | Enforced job lifecycle: `queued → processing → completed/failed → retrying` |
| **Workers** | Concurrent RabbitMQ consumers with panic recovery, cancellation propagation |
| **Rate Limiting** | Redis-backed sliding window per user/IP |
| **Observability** | Prometheus metrics + Grafana dashboards (HTTP latency, job throughput, queue depth) |
| **Frontend** | React + TypeScript dashboard with live polling, progress bars, streaming logs |
| **Infrastructure** | Docker Compose, Kubernetes manifests (HPA, Ingress, health probes), Terraform (AWS) |

## Tech Stack

- **Backend:** Go 1.26, Chi, pgx (raw SQL), golang-migrate, JWT, slog
- **Frontend:** React 18, TypeScript, Vite, TanStack Query, Axios, Vanilla CSS
- **Data:** PostgreSQL 17, Redis 7, RabbitMQ 4
- **DevOps:** Docker, Kubernetes, Terraform, Prometheus, Grafana
- **Testing:** Go testing, httptest, k6 load testing

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 22+
- Docker & Docker Compose
- PostgreSQL 17, RabbitMQ, Redis (or use Docker Compose)

### Option 1: Docker Compose (Recommended)

```bash
# Start everything — Postgres, RabbitMQ, Redis, API, Worker (x2), Frontend
docker compose up -d

# View logs
docker compose logs -f api
docker compose logs -f worker

# Access the app
open http://localhost:3000
```

### Option 2: Local Development

```bash
# 1. Start infrastructure
docker compose up -d postgres rabbitmq redis

# 2. Run migrations
make migrate-up

# 3. Start API server
cp backend/.env.example backend/.env
make run-api

# 4. Start worker
make run-worker

# 5. Start frontend dev server
cd frontend && npm install && npm run dev
```

## Project Structure

```
├── backend/
│   ├── cmd/
│   │   ├── api/              # API server entrypoint
│   │   └── worker/           # Worker entrypoint
│   ├── internal/
│   │   ├── config/           # Environment-based configuration
│   │   ├── database/         # PostgreSQL connection pool
│   │   ├── handler/          # HTTP handlers
│   │   ├── metrics/          # Prometheus metric definitions
│   │   ├── middleware/       # RequestID, Logger, Auth, RateLimit, Metrics
│   │   ├── models/           # Domain models + state machine
│   │   ├── queue/            # RabbitMQ connection + outbox publisher
│   │   ├── redis/            # Redis client
│   │   ├── repository/       # Data access (raw SQL + pgx)
│   │   ├── router/           # Chi route definitions
│   │   ├── service/          # Business logic
│   │   ├── validator/        # Input validation
│   │   └── worker/           # Job processors + consumer
│   ├── migrations/           # SQL migration files
│   └── pkg/response/         # HTTP response helpers
├── frontend/
│   └── src/
│       ├── api/              # Axios clients
│       ├── components/       # Reusable UI components
│       └── pages/            # Login, Register, Dashboard, Jobs, JobDetail
├── deploy/
│   └── kubernetes/           # Full K8s manifest set
│       ├── api/              # API Deployment + Service + Migration Job
│       ├── worker/           # Worker Deployment
│       ├── postgres/         # StatefulSet + PVC
│       ├── rabbitmq/         # StatefulSet + PVC
│       ├── redis/            # Deployment
│       ├── frontend/         # Deployment + Service
│       ├── prometheus/       # Deployment + ConfigMap + RBAC
│       ├── grafana/          # Deployment + Dashboards
│       ├── ingress.yaml      # Nginx Ingress
│       ├── hpa.yaml          # HPA for API + Worker
│       └── kustomization.yaml
├── terraform/                # AWS infrastructure (EKS, RDS, ElastiCache, MQ, ECR)
├── docker-compose.yml
└── Makefile
```

## API Reference

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/auth/register` | Register a new user |
| `POST` | `/api/v1/auth/login` | Login and receive JWT |

### Jobs (Authenticated)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/jobs` | Create a job (requires `Idempotency-Key` header) |
| `GET` | `/api/v1/jobs` | List your jobs (paginated, filterable) |
| `GET` | `/api/v1/jobs/{id}` | Get job details |
| `POST` | `/api/v1/jobs/{id}/cancel` | Cancel a queued job |
| `GET` | `/api/v1/jobs/{id}/logs` | Get execution logs |
| `GET` | `/api/v1/jobs/stats` | Get job statistics |

### Health & Metrics

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health/live` | Liveness probe (always 200 if process runs) |
| `GET` | `/health/ready` | Readiness probe (checks Postgres + Redis) |
| `GET` | `/metrics` | Prometheus metrics |

## Job Types

| Type | Description |
|------|-------------|
| `report_generation` | Simulates generating a large report |
| `data_processing` | Simulates batch data transformation |
| `image_processing` | Registered but not yet implemented |
| `notification` | Registered but not yet implemented |

## Job State Machine

```
               ┌───────────┐
               │  QUEUED    │──────────────┐
               └─────┬─────┘              │
                     │                     │
                     ▼                     ▼
               ┌───────────┐        ┌───────────┐
          ┌───▶│PROCESSING │        │ CANCELLED │
          │    └──┬──────┬──┘       └───────────┘
          │       │      │
          │       ▼      ▼
          │ ┌─────────┐ ┌───────────┐
          │ │COMPLETED│ │  FAILED   │
          │ └─────────┘ └─────┬─────┘
          │                   │
          │                   ▼
          │             ┌───────────┐
          └─────────────│ RETRYING  │
                        └───────────┘
```

## Testing

```bash
# Run all unit tests
make test

# Run with race detector
cd backend && go test -race ./...

# Load testing (requires k6: https://k6.io)
k6 run backend/loadtest.js

# Load test against a different host
k6 run backend/loadtest.js --env BASE_URL=https://staging.example.com
```

## Kubernetes Deployment

```bash
# Build images
docker build -f backend/Dockerfile.api -t jobplatform-api:latest backend/
docker build -f backend/Dockerfile.worker -t jobplatform-worker:latest backend/
docker build -f frontend/Dockerfile -t jobplatform-frontend:latest frontend/

# Deploy entire stack
kubectl apply -k deploy/kubernetes/

# Check status
kubectl -n jobplatform get pods

# Scale workers
kubectl -n jobplatform scale deployment/worker --replicas=5

# View API logs
kubectl -n jobplatform logs -l app=api -f
```

## Terraform (AWS)

```bash
cd terraform/
terraform init
terraform plan -var-file=dev.tfvars
terraform apply -var-file=dev.tfvars

# Configure kubectl for the new cluster
$(terraform output -raw kubectl_config_command)
```

## Observability

- **Prometheus**: Scrapes `/metrics` from API (`:8080`) and Worker (`:8081`)
- **Grafana**: Pre-provisioned dashboard at `http://localhost:3000` (in K8s)
- **Metrics tracked**:
  - `jobplatform_http_requests_total` — HTTP request count by method/path/status
  - `jobplatform_http_request_duration_seconds` — Request latency histogram
  - `jobplatform_http_requests_in_flight` — Active concurrent requests
  - `jobplatform_jobs_created_total` — Jobs created by type
  - `jobplatform_jobs_processed_total` — Jobs processed by type and outcome
  - `jobplatform_jobs_processing_duration_seconds` — End-to-end processing time
  - `jobplatform_jobs_in_flight` — Active worker job count
  - `jobplatform_outbox_published_total` — Outbox events published
  - `jobplatform_outbox_publish_errors_total` — Outbox publish failures

## Key Engineering Decisions

1. **Transactional Outbox Pattern**
   I wanted to ensure that if a user creates a job, it is guaranteed to eventually execute, even if RabbitMQ crashes at that exact millisecond. The API writes the job and its outbox event in a single PostgreSQL transaction. A separate outbox publisher routine then forwards pending events to RabbitMQ. This provides at-least-once delivery without the complexity of two-phase commit.

2. **Raw SQL over an ORM**
   I used `pgx` instead of an ORM like GORM. While ORMs are fast for prototyping, they often obscure what the database is actually doing and make complex joins or database-specific locks (like `FOR UPDATE SKIP LOCKED` which the outbox pattern needs) harder to write. Writing raw, parameterized SQL kept the data access layer completely transparent.

3. **State Machine Enforcement**
   All job status transitions (`queued -> processing -> completed`) are strictly validated in the domain layer (`models.ValidateTransition`). When updating the database, we use `WHERE status = $expected` to implement optimistic concurrency control. This prevents a slow worker from overriding a cancelled job status.

4. **Idempotency**
   To prevent duplicate jobs if a client retries a failed HTTP request, the API requires an `Idempotency-Key` header. We store this key in Postgres along with the job record inside the transaction. If the same key is submitted again, we return the cached response rather than creating a duplicate.

5. **Worker Cancellation Propagation**
   Workers need to be able to stop mid-execution if the user cancels a job. Go's `context.Context` is perfect for this. The worker polls the database every 5 seconds; if the job status changes to cancelled, the context is cancelled, which propagates down to the executing `JobProcessor` to abort safely without leaking goroutines.

6. **Rate Limiting (Fail Open)**
   The API uses Redis to rate-limit requests per user. However, I designed it to "fail open" — if Redis goes down, the API logs a warning but continues serving traffic. It's better to temporarily lose rate-limiting than to take down the entire API because a cache node crashed.

## License

MIT
#   T a s k F l o w 
 
 
