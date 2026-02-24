# Parallelis: Distributed Execution Engine

A high-performance distributed job processing system built in **Go**, designed to handle thousands of concurrent requests with zero job loss guarantees. The system demonstrates real-world distributed systems patterns including message queuing, rate limiting, caching, automatic retry with exponential backoff, and horizontal auto-scaling on AWS.

## Performance Benchmarks

| Metric | Result |
|--------|--------|
| Peak Throughput | **4,043 requests/sec** |
| Total Requests Handled | **5.1M+ requests** with **0% failure rate** |
| Avg Response Time (under load) | **103ms** (p50), **150ms** (p95) |
| Auto-scaling Response | 2 → 4 instances in **< 5 minutes** |
| Job Processing Capacity | **1,000+ jobs/minute** |
| Cache Hit Performance | **2ms avg** vs 10ms DB query (80% hit rate) |
| Rate Limiting | **100 req/min per client** via Redis token bucket |
| Resilience | **0 failures** after killing 2/3 running instances mid-load |

## Architecture

```
Client → API (Gin) → PostgreSQL (PENDING) → Scheduler → Kafka (16 partitions) → Workers → PostgreSQL (COMPLETED)
           ↓                                                                        ↓
     Redis (rate limit)                                                       Redis (cache)
           ↓
    ALB (load balancing) → ECS Fargate (auto-scaled 2-4 instances)
```

### Request Flow
1. Client submits job via REST API with rate limiting (token bucket algorithm)
2. Job persisted to PostgreSQL in `PENDING` status — source of truth
3. Scheduler polls DB every 5s, publishes job IDs to Kafka
4. Workers consume from Kafka, fetch job details (Redis cache-aside pattern)
5. Workers process job, update status to `COMPLETED`, acknowledge Kafka offset
6. On failure: exponential backoff retry (2^n seconds), max 3 retries → dead letter queue

## Key Technical Decisions

### Why Go?
- **Goroutine-based concurrency** — lightweight concurrent workers consuming from Kafka partitions without thread pool overhead
- **Low-latency networking** — sub-millisecond overhead for HTTP and Kafka operations
- **Compiled binary** — single static binary deployment to ECS Fargate, no JVM warmup
- **Memory efficiency** — ~50MB per container vs 256MB+ for JVM-based services

### Kafka Configuration (Optimized for Throughput + Durability)
- **16 partitions** — enables up to 16 parallel workers
- **acks=all** — wait for all replicas to acknowledge (zero message loss)
- **Manual offset commit** — only after successful DB update (at-least-once delivery)
- **Gzip compression** — reduced network bandwidth, Alpine-compatible
- **Client ID partitioning** — jobs from same client route to same partition (ordering guarantee)

### Redis Dual-Purpose Design
- **Rate Limiting (Token Bucket):** 100 requests/min per client, prevents flash sale abuse, fail-open if Redis unavailable
- **Cache-Aside Pattern:** 15-min TTL on job details, reduces PostgreSQL load by ~80% during traffic spikes, invalidate-on-write consistency

### Retry Strategy (Exponential Backoff)
```
Attempt 1 fails → retry in 2s
Attempt 2 fails → retry in 4s  
Attempt 3 fails → retry in 8s
Attempt 4 → DEAD_LETTER (manual intervention)
```

## Infrastructure & Scaling

Deployed on **AWS ECS Fargate** with infrastructure-as-code via **Terraform**.

### Auto-Scaling Configuration
| Parameter | Value |
|-----------|-------|
| Min Instances | 2 |
| Max Instances | 4 (tested up to 6) |
| CPU Target | 70% average utilization |
| Scale-out Cooldown | 60-300 seconds (experimented) |
| Load Balancer | Application Load Balancer (ALB) |
| Health Check | `/health` endpoint, 30s interval |

### Scaling Behavior Under Load
| Load | Instances | Avg CPU | Avg Response |
|------|-----------|---------|-------------|
| 200 users | 2 → 3 | 86% → 60% | 106ms |
| 500 users | 3 → 4 | 99% → 75% | 117ms |
| 500 users (4 instances) | 4 | 75% | 103ms |

### Resilience Testing
- Stopped 2 of 3 running instances during active load test
- ALB immediately routed traffic to remaining healthy instance
- ECS auto-launched replacement tasks within 30 seconds
- **Zero request failures** across 2.95M requests during failure scenario

## Load Testing

Load testing performed with **Locust** using `FastHttpUser` for maximum throughput.

### Single Instance Breaking Point (Part II)
```
500 concurrent users → CPU: 100% → Avg: 274ms → p95: 620ms → 1,809 RPS
```

### Horizontally Scaled (Part III)  
```
500 concurrent users → CPU: 75% → Avg: 117ms → p95: 200ms → 3,368 RPS
```

**Result:** 2x throughput improvement, 57% reduction in average latency, automatic scaling with zero downtime.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.21+ |
| HTTP Framework | Gin |
| ORM | GORM (PostgreSQL) |
| Message Queue | Apache Kafka (segmentio/kafka-go) |
| Cache & Rate Limit | Redis (go-redis/v9) |
| Database | PostgreSQL |
| Infrastructure | AWS ECS Fargate, ALB, Terraform |
| Load Testing | Locust (Python, FastHttpUser) |
| Containerization | Docker (multi-stage Alpine build) |

## Project Structure

```
├── src/main/go/com/demo/jobprocessor/
│   ├── config/                 # Kafka, Redis configuration
│   │   ├── kafka_consumer_config.go
│   │   ├── kafka_producer_config.go
│   │   └── redis_config.go
│   ├── controller/             # REST API endpoints
│   │   └── job_controller.go
│   ├── dto/                    # Request/Response objects
│   │   ├── job_request.go
│   │   └── job_response.go
│   ├── exception/              # Error handling
│   │   ├── error_response.go
│   │   ├── global_exception_handler.go
│   │   └── job_not_found.go
│   ├── model/                  # Domain models
│   │   ├── job.go
│   │   ├── job_status.go
│   │   └── job_type.go
│   ├── repository/             # Data access layer
│   │   └── job_repository.go
│   ├── service/                # Business logic
│   │   ├── cache_service.go
│   │   ├── job_scheduler.go
│   │   ├── job_service.go
│   │   ├── job_worker.go
│   │   └── rate_limit_service.go
│   └── main.go                 # Application entry point
├── terraform/                  # Infrastructure as code
├── experiment_partitions.py    # Load testing experiments
├── go.mod
└── README.md
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/jobs` | Create job (202 Accepted) |
| GET | `/api/jobs/:id` | Get job by ID |
| GET | `/api/jobs?clientId=X` | Get jobs by client |
| GET | `/api/jobs/stats` | System statistics |
| GET | `/api/jobs/health` | Health check |

### Rate Limiting Headers
```
X-Client-Id: customer-12345      (required)
X-RateLimit-Limit: 100           (response)
X-RateLimit-Remaining: 87        (response)
```

## Running Locally

```bash
# Prerequisites: Go 1.21+, Docker, Kafka, Redis, PostgreSQL

# Start dependencies
docker-compose up -d kafka redis postgres

# Run the application
go run main.go

# Run tests
go test ./...

# Load test
locust -f experiment_partitions.py --host=http://localhost:8080
```

## Distributed Systems Concepts Demonstrated

- **Message Queue Semantics:** At-least-once delivery with manual Kafka offset management
- **Cache-Aside Pattern:** Redis caching with write-through invalidation
- **Token Bucket Rate Limiting:** Per-client fairness during flash sale scenarios
- **Exponential Backoff:** Intelligent retry for transient failures
- **Horizontal Auto-Scaling:** CPU-based target tracking with ALB health checks
- **Graceful Degradation:** Fail-open rate limiting, circuit breaking on cache failures
- **Zero-Downtime Deployments:** Rolling updates via ECS with ALB draining