# Go Banking Simulation API - Complete Setup & Usage Guide

## 📋 Table of Contents
1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start - Docker Setup](#quick-start---docker-setup)
4. [Environment Configuration](#environment-configuration)
5. [Database Setup & Migrations](#database-setup--migrations)
6. [Implemented Features](#implemented-features)
7. [API Endpoints](#api-endpoints)
8. [Testing with Postman](#testing-with-postman)
9. [Monitoring & Observability](#monitoring--observability)
10. [Architecture Overview](#architecture-overview)
11. [Troubleshooting](#troubleshooting)

---

## 🎯 Overview

This is a comprehensive **Go Banking Simulation API** built with modern Go practices, featuring:
- **User Management**: Registration, authentication, RBAC
- **Financial Operations**: Credits, debits, transfers with ACID compliance
- **Advanced Features**: Scheduled transactions, multi-currency, audit trails
- **Enterprise Features**: Event sourcing, monitoring, caching, circuit breakers
- **Production Ready**: Docker, monitoring stack, database replication

---

## 📋 Prerequisites

- **Docker & Docker Compose** (latest versions)
- **Git** for cloning the repository
- **PowerShell/Bash** for running scripts
- **Optional**: Postman for API testing

---

## 🚀 Quick Start - Docker Setup

### Step 1: Clone & Navigate
```bash
git clone <your-repo-url>
cd InsiderProject2
```

### Step 2: Environment Setup
```bash
# Copy and configure environment (optional - defaults will work)
cp .env.example .env  # If you have this file
```

### Step 3: Launch All Services
```bash
docker compose -f docker-compose.dev.yml build
# Start all services (PostgreSQL, Redis, App, Nginx, Monitoring)
docker compose -f docker-compose.dev.yml up -d
```

### Step 4: Verify Services Are Running
```bash
# Check all containers are healthy
docker compose -f docker-compose.dev.yml ps

# Check app health
curl http://localhost:8080/healthz
```

### Step 5: Run Database Migrations
```bash
# Run migrations inside the running container
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /migrations/001_create_users.up.sql
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /migrations/002_create_balances.up.sql
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /migrations/003_create_transactions.up.sql
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /migrations/004_create_audit_logs.up.sql
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /migrations/005_fix_balances_trigger.up.sql
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /migrations/006_add_is_active_to_users.up.sql
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /migrations/007_create_events.up.sql
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /migrations/008_add_currency_support.up.sql
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /migrations/009_create_scheduled_transactions.up.sql
```

### Step 6: Seed Initial Data (Optional)
```bash
# Load seed data
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -f /scripts/seed.sql
```

### Step 7: Verify API is Working
```bash
# Test basic connectivity
curl http://localhost:8080/api/v1/ping

# Test metrics endpoint
curl http://localhost:8080/metrics/basic
```

---

## ⚙️ Environment Configuration

The application uses these environment variables (configured in `docker-compose.dev.yml`):

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_URL` | `postgres://postgres:postgres@db:5432/banking_sim?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | `your-super-secret-jwt-key-change-in-production` | JWT signing secret |
| `PORT` | `8080` | Application port |
| `ENV` | `dev` | Environment (dev/prod) |
| `ALLOWED_ORIGINS` | `*` | CORS allowed origins |

---

## 🗄️ Database Setup & Migrations

### Database Schema Overview

The system uses **9 migration files** creating these core tables:

1. **`users`** - User accounts with RBAC
2. **`balances`** - User account balances with currency support
3. **`transactions`** - All financial transactions
4. **`audit_logs`** - Complete audit trail
5. **`events`** - Event sourcing data
6. **`scheduled_transactions`** - Future transaction scheduling

### Key Relationships
- `balances.user_id → users.id`
- `transactions.from_user_id|to_user_id → users.id`
- `audit_logs.entity_id → various entities`
- `events.aggregate_id → various entities`

---

## ✨ Implemented Features

### ✅ Core Banking Features (100% Complete)

#### 🔐 Authentication & Authorization
- **User Registration** with email/username validation
- **JWT-based Authentication** (access + refresh tokens)
- **Role-Based Access Control** (admin/user roles)
- **Token Refresh** mechanism
- **Password Security** with bcrypt hashing

#### 💰 Financial Operations
- **Credit Transactions** - Add money to account
- **Debit Transactions** - Remove money (balance validation)
- **Transfer Operations** - Atomic money transfers between users
- **Transaction Rollback** - Compensating transactions
- **Balance Tracking** - Real-time balance updates
- **Multi-Currency Support** - USD, EUR, etc.

#### 📊 Balance Management
- **Current Balance** - Real-time account balance
- **Historical Balances** - Balance snapshots over time
- **Point-in-Time Balance** - Balance at specific timestamp
- **Balance Reconciliation** - Audit trail verification

#### ⏰ Scheduled Transactions
- **Future Transaction Scheduling** - One-time or recurring
- **Automatic Execution** - Background worker processing
- **Schedule Management** - Create, list, cancel operations

### ✅ Enterprise Features

#### 📈 Monitoring & Observability
- **Prometheus Metrics** - Application and business metrics
- **Grafana Dashboards** - Real-time visualization
- **Distributed Tracing** - Jaeger integration
- **Structured Logging** - JSON logs with correlation IDs
- **Health Checks** - Application and database health

#### ⚡ Performance & Reliability
- **Worker Pools** - Async transaction processing
- **Redis Caching** - Hot data caching
- **Database Replication** - Primary-replica setup
- **Circuit Breakers** - Fault tolerance with automatic failure detection
- **Rate Limiting** - Request throttling
- **Connection Pooling** - Database connection management

#### 🔍 Audit & Compliance
- **Complete Audit Trail** - All operations logged
- **Event Sourcing** - Immutable event log
- **Transaction History** - Comprehensive filtering
- **Access Logging** - Request/response logging

### ✅ Advanced Architecture

#### 🏗️ Clean Architecture
- **Domain Layer** - Business entities and rules
- **Repository Layer** - Data persistence
- **Service Layer** - Business logic orchestration
- **API Layer** - HTTP request handling
- **Middleware Layer** - Cross-cutting concerns

#### 🔄 Event-Driven Architecture
- **Event Sourcing** - State from events
- **Projector Workers** - Event materialization
- **Background Processing** - Async operations
- **Message Queues** - Go channels for job processing

---

## 🌐 API Endpoints

### Base URL: `http://localhost:8080/api/v1`

### 🔐 Authentication Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `POST` | `/auth/register` | User registration | ❌ |
| `POST` | `/auth/login` | User login | ❌ |
| `POST` | `/auth/refresh` | Refresh access token | ❌ |

### 👥 User Management (Admin Only)

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `GET` | `/users` | List all users | ✅ (Admin) |
| `GET` | `/users/{id}` | Get user by ID | ✅ (Admin) |
| `PUT` | `/users/{id}` | Update user | ✅ (Admin) |
| `DELETE` | `/users/{id}` | Delete user | ✅ (Admin) |

### 💰 Balance Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `GET` | `/balances/current` | Get current balance | ✅ |
| `GET` | `/balances/historical` | Get balance history | ✅ |
| `GET` | `/balances/at-time?timestamp=...` | Get balance at specific time | ✅ |

### 💸 Transaction Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `POST` | `/transactions/credit` | Credit money to account | ✅ |
| `POST` | `/transactions/debit` | Debit money from account | ✅ |
| `POST` | `/transactions/transfer` | Transfer money between users | ✅ |
| `POST` | `/transactions/{id}/rollback` | Rollback a transaction | ✅ |
| `GET` | `/transactions/{id}` | Get transaction details | ✅ |
| `GET` | `/transactions/history` | Get transaction history | ✅ |

### ⏰ Scheduled Transaction Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `POST` | `/scheduled-transactions` | Schedule a transaction | ✅ |
| `GET` | `/scheduled-transactions` | List scheduled transactions | ✅ |
| `GET` | `/scheduled-transactions/{id}` | Get scheduled transaction | ✅ |
| `DELETE` | `/scheduled-transactions/{id}` | Cancel scheduled transaction | ✅ |

### 📊 Monitoring Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `GET` | `/healthz` | Health check | ❌ |
| `GET` | `/metrics` | Prometheus metrics | ❌ |
| `GET` | `/metrics/basic` | Basic metrics (JSON) | ❌ |
| `GET` | `/api/v1/metrics/circuit-breakers` | Circuit breaker status | ❌ |

### 🛡️ Circuit Breaker Test Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| `GET` | `/api/v1/test/circuit-breaker/success` | Test successful requests | ❌ |
| `GET` | `/api/v1/test/circuit-breaker/failure` | Test failure handling | ❌ |
| `GET` | `/api/v1/test/circuit-breaker/timeout` | Test timeout handling | ❌ |

---

## 🛡️ Circuit Breaker Testing Guide

Circuit breakers protect your API from cascading failures by automatically stopping requests to failing services. When a service fails repeatedly, the circuit breaker "opens" and rejects subsequent requests immediately, preventing system overload.

### How Circuit Breakers Work

1. **Closed State**: Normal operation - requests pass through
2. **Open State**: Service is failing - requests are rejected immediately with 503 status
3. **Half-Open State**: Testing recovery - allows limited requests to check if service recovered

### Circuit Breaker Configuration

Each service has configurable thresholds:
- **Failure Threshold**: Number of failures before opening (default: 2-5 failures)
- **Reset Timeout**: Time to wait before testing recovery (default: 10-60 seconds)
- **Call Timeout**: Maximum time for individual requests (default: 30 seconds)

### Testing Circuit Breaker Behavior

#### 1. Monitor Circuit Breaker Status
```bash
# Check all circuit breakers status
curl http://localhost:8080/api/v1/metrics/circuit-breakers

# Expected response:
{
  "circuit_breakers": {
    "test-success-service": {
      "state": "closed",
      "total_requests": 0,
      "total_failures": 0,
      "total_successes": 0,
      "current_failures": 0
    },
    "test-failure-service": {
      "state": "closed",
      "total_requests": 0,
      "total_failures": 0,
      "total_successes": 0,
      "current_failures": 0
    },
    "test-timeout-service": {
      "state": "closed",
      "total_requests": 0,
      "total_failures": 0,
      "total_successes": 0,
      "current_failures": 0
    }
  }
}
```

#### 2. Test Successful Requests
```bash
# This should always work and keep circuit breaker closed
curl http://localhost:8080/api/v1/test/circuit-breaker/success

# Expected response:
{"message":"Circuit breaker test - success","status":"ok"}
```

#### 3. Test Failure Handling
```bash
# First request - will fail but circuit breaker stays closed
curl http://localhost:8080/api/v1/test/circuit-breaker/failure

# Expected response:
{"error":"Circuit breaker test - simulated failure","code":500}

# Second request - circuit breaker opens after 2 failures
curl http://localhost:8080/api/v1/test/circuit-breaker/failure

# Expected response:
{"error":"Service temporarily unavailable","code":503,"service":"test-failure-service"}
```

#### 4. Verify Circuit Breaker Opened
```bash
# Check status - should now be "open"
curl http://localhost:8080/api/v1/metrics/circuit-breakers

# Expected: test-failure-service state = "open"
```

#### 5. Test Fast-Fail Behavior
```bash
# Additional requests while open - all return 503 immediately
curl http://localhost:8080/api/v1/test/circuit-breaker/failure

# Expected response (immediate):
{"error":"Service temporarily unavailable","code":503,"service":"test-failure-service"}
```

#### 6. Test Recovery (Half-Open State)
```bash
# Wait ~10 seconds for reset timeout
sleep 10

# Circuit breaker enters half-open state and allows one test request
curl http://localhost:8080/api/v1/test/circuit-breaker/failure

# If the service recovered: circuit closes, request succeeds
# If still failing: circuit stays open, request fails
```

### Circuit Breaker Metrics Explained

| Metric | Description |
|--------|-------------|
| `state` | Current state: "closed", "open", or "half-open" |
| `total_requests` | Total number of requests made |
| `total_failures` | Total number of failed requests |
| `total_successes` | Total number of successful requests |
| `current_failures` | Current consecutive failure count |

### Testing with Different Configurations

#### Low Threshold (Opens Quickly)
```bash
# Circuit breaker opens after just 1 failure
curl -X POST http://localhost:8080/api/v1/test/circuit-breaker/failure
# Opens immediately after first failure
```

#### High Threshold (More Tolerant)
```bash
# Would need 5 failures to open (if configured)
# Useful for services with occasional failures
```

### Production Circuit Breaker Usage

In production, circuit breakers protect critical external services:

```bash
# Example: External payment service
curl http://localhost:8080/api/v1/transactions/credit

# If external payment service fails repeatedly:
# 1. Circuit breaker opens after threshold
# 2. All payment requests return 503 immediately
# 3. System remains stable, no cascading failures
# 4. Manual intervention or automatic recovery possible
```

### Best Practices

1. **Monitor Circuit Breaker States** - Set up alerts when breakers open
2. **Configure Appropriate Thresholds** - Balance between protection and availability
3. **Test Failure Scenarios** - Regularly test circuit breaker behavior
4. **Implement Fallbacks** - Have alternative paths when services are unavailable
5. **Log Circuit Breaker Events** - Track when breakers open/close for analysis

---

## 🔴 Redis Cache Testing Guide

### Overview
The Postman collection includes a dedicated **"🔴 REDIS CACHE TESTING"** folder with comprehensive Redis cache testing scenarios. This section tests:

- **Cache Hits/Misses** - Verify data is served from Redis cache
- **Cache Invalidation** - Test that updates clear relevant caches
- **Rate Limiting** - Test Redis-based rate limiting functionality
- **Cache Warm-up** - Test automatic cache population after invalidation
- **Performance Testing** - Measure cache vs database response times

### Cache TTL Configuration
- **Users**: 30 minutes
- **Balances**: 10 minutes
- **Transactions**: 15 minutes
- **Rate Limits**: 1 minute

### Redis Testing Workflow

#### Step 1: Setup & Authentication
1. **Login** with admin credentials to get access token
2. **Set Variables**:
   - `{{IdToGet}}` - User ID to test cache hits
   - `{{idToUpdateUser}}` - User ID for update testing
   - `{{TransactionId}}` - Transaction ID for testing

#### Step 2: Test Cache Statistics
```bash
GET {{base_url}}/metrics/basic
```
- Shows current cache statistics
- Displays Redis connection status
- Includes rate limiting counters

#### Step 3: Test User Cache Behavior
1. **First Request** (Cache Miss):
   ```bash
   GET {{base_url}}/users/{{IdToGet}}
   Authorization: Bearer {{accessToken}}
   ```
   - Hits database, caches result
   - Expected: ~50-200ms response time

2. **Second Request** (Cache Hit):
   ```bash
   GET {{base_url}}/users/{{IdToGet}}
   Authorization: Bearer {{accessToken}}
   ```
   - Serves from Redis cache
   - Expected: <10ms response time ⚡

#### Step 4: Test Cache Invalidation
1. **Update User**:
   ```bash
   PUT {{base_url}}/users/{{idToUpdateUser}}
   Authorization: Bearer {{accessToken}}
   {
       "username": "cache_test_user",
       "email": "cache.test@example.com"
   }
   ```
   - Invalidates user cache

2. **Verify Invalidation**:
   ```bash
   GET {{base_url}}/users/{{idToUpdateUser}}
   Authorization: Bearer {{accessToken}}
   ```
   - Should hit database again (slower response)

#### Step 5: Test Balance Cache
```bash
GET {{base_url}}/balances/current
Authorization: Bearer {{accessToken}}
```
- Balance cached with 10-minute TTL
- Multiple requests should be fast after first call

#### Step 6: Test Transaction Cache
```bash
GET {{base_url}}/transactions/{{TransactionId}}
Authorization: Bearer {{accessToken}}
```
- Transaction cached with 15-minute TTL
- Subsequent requests serve from cache

#### Step 7: Test Rate Limiting
Run the **"Test Rate Limiting"** request multiple times:
```bash
POST {{base_url}}/auth/login
# Run 6+ times rapidly
```
- First 5 requests: ✅ 200 OK
- 6th request: ❌ 429 Rate Limited
- Wait 1 minute or reset counter

#### Step 8: Test Cache Warm-up
1. **Create Credit Transaction**:
   ```bash
   POST {{base_url}}/transactions/credit
   Authorization: Bearer {{accessToken}}
   {
       "amount": 50.00,
       "currency": "USD"
   }
   ```
   - Invalidates balance cache

2. **Check Balance Again**:
   ```bash
   GET {{base_url}}/balances/current
   Authorization: Bearer {{accessToken}}
   ```
   - Hits database, recaches updated balance

### Performance Testing

#### Cache Performance Measurement
Run balance requests multiple times and observe response times:
- **First request**: Database hit (~50-200ms) 🐌
- **Subsequent requests**: Cache hits (<10ms) 🚀

#### Cache Hit Indicators
- Response time < 10ms = Likely cache hit
- Response time 10-100ms = Possible cache hit
- Response time > 100ms = Likely database hit

### Expected Test Results

| Test Scenario | Expected Behavior | Success Indicators |
|---------------|-------------------|-------------------|
| Cache Hit | Fast response (<10ms) | ⚡ Sub-10ms response |
| Cache Miss | Slower response | 🐌 50-200ms response |
| Cache Invalidation | Next request slow | Cache cleared, DB hit |
| Rate Limiting | 429 after threshold | 5/5 → 429 on 6th |
| Cache Warm-up | Fast after invalidation | Auto-recaching works |

### Troubleshooting Cache Tests

#### Cache Not Working
- Check Redis connection: `docker logs insiderproject2-redis-1`
- Verify cache service injection: Check application logs
- Test Redis directly: `docker exec insiderproject2-redis-1 redis-cli ping`

#### Rate Limiting Not Working
- Check rate limit counters: `GET /metrics/basic`
- Verify middleware is applied: Check router logs
- Test with different IPs if needed

#### Slow Responses
- Database connection issues
- Redis connection problems
- High server load
- Network latency

---

## 🧪 Testing with Postman

### Import the Collection
1. Open Postman
2. Import `INSIDERPROJECT.postman_collection.json`
3. Set environment variable:
   - `base_url` = `http://localhost:8080`

### Test Flow Example

#### 1. Register a New User
```json
POST {{base_url}}/auth/register
{
    "username": "johndoe",
    "email": "john.doe@example.com",
    "password": "password123",
    "role": "user"
}
```

#### 2. Login (Auto-saves tokens)
```json
POST {{base_url}}/auth/login
{
    "email": "john.doe@example.com",
    "password": "password123"
}
```

#### 3. Check Current Balance
```json
GET {{base_url}}/balances/current
Authorization: Bearer {{accessToken}}
```

#### 4. Credit Money
```json
POST {{base_url}}/transactions/credit
Authorization: Bearer {{accessToken}}
{
    "amount": 1000.00,
    "currency": "USD"
}
```

#### 5. Transfer Money
```json
POST {{base_url}}/transactions/transfer
Authorization: Bearer {{accessToken}}
{
    "to_user_id": "target-user-uuid",
    "amount": 100.00,
    "currency": "USD"
}
```

#### 6. View Transaction History
```json
GET {{base_url}}/transactions/history?limit=10&type=transfer
Authorization: Bearer {{accessToken}}
```

---

## 📊 Monitoring & Observability

### Accessing Monitoring Tools

#### Grafana Dashboard
- **URL**: http://localhost:3000
- **Username**: `admin`
- **Password**: `admin123`
- **Dashboard**: "Banking Simulation Overview"

#### Prometheus
- **URL**: http://localhost:9090
- **Metrics Endpoint**: http://localhost:8080/metrics

#### Jaeger Tracing
- **URL**: http://localhost:16686

### Key Metrics Available
- **HTTP Request Metrics**: Response times, status codes, request counts
- **Business Metrics**: Transaction counts, balance changes, user activity
- **System Metrics**: Goroutines, memory usage, queue depths
- **Database Metrics**: Connection pool status, query performance
- **Worker Pool Metrics**: Active workers, queued jobs, processing times
- **Circuit Breaker Metrics**: Service states, failure counts, recovery status

---

## 🏗️ Architecture Overview

### System Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Nginx Proxy   │    │   Go Application│    │   PostgreSQL    │
│   (Port 8080)   │◄──►│   (Port 8080)   │◄──►│   Primary DB    │
│                 │    │                 │    │   (Port 5432)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Prometheus    │    │     Redis       │    │ PostgreSQL      │
│   (Port 9090)   │    │   (Port 6379)   │    │   Replica DB    │
│                 │    │                 │    │   (Port 5433)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌─────────────────┐
│    Grafana      │    │     Jaeger      │
│   (Port 3000)   │    │   (Port 16686)  │
│                 │    │                 │
└─────────────────┘    └─────────────────┘
```

### Data Flow

1. **HTTP Request** → Nginx → Go Application
2. **Authentication** → JWT validation → User context
3. **Business Logic** → Service layer orchestration
4. **Data Persistence** → Repository layer → PostgreSQL
5. **Async Processing** → Worker pools → Job queues
6. **Event Processing** → Event sourcing → Projectors
7. **Caching** → Redis for hot data
8. **Monitoring** → Prometheus metrics → Grafana dashboards

---

## 🔧 Troubleshooting

### Common Issues & Solutions

#### 1. Database Connection Issues
```bash
# Check if PostgreSQL is running
docker compose -f docker-compose.dev.yml ps db

# Check database logs
docker compose -f docker-compose.dev.yml logs db

# Test database connection
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim -c "SELECT 1;"
```

#### 2. Application Startup Issues
```bash
# Check application logs
docker compose -f docker-compose.dev.yml logs app

# Check health endpoint
curl http://localhost:8080/healthz

# Verify environment variables
docker compose -f docker-compose.dev.yml exec app env
```

#### 3. Migration Issues
```bash
# Run migrations manually
docker compose -f docker-compose.dev.yml exec db bash
psql -U postgres -d banking_sim -f /migrations/001_create_users.up.sql

# Check migration status
psql -U postgres -d banking_sim -c "\dt"
```

#### 4. Redis Connection Issues
```bash
# Check Redis status
docker compose -f docker-compose.dev.yml ps redis

# Test Redis connection
docker compose -f docker-compose.dev.yml exec redis redis-cli ping
```

#### 5. Port Conflicts
```bash
# Check what's using ports
netstat -tulpn | grep -E ':(8080|5432|6379|9090|3000|16686)'

# Stop conflicting services or change ports in docker-compose.dev.yml
```

#### 6. Circuit Breaker Issues
```bash
# Check circuit breaker status
curl http://localhost:8080/api/v1/metrics/circuit-breakers

# Reset circuit breaker state (requires application restart)
docker compose -f docker-compose.dev.yml restart app

# Test circuit breaker behavior
curl http://localhost:8080/api/v1/test/circuit-breaker/failure
curl http://localhost:8080/api/v1/test/circuit-breaker/success

# Monitor circuit breaker logs
docker compose -f docker-compose.dev.yml logs app | grep -i circuit
```

### Useful Commands

#### Docker Management
```bash
# View all running services
docker compose -f docker-compose.dev.yml ps

# View logs for specific service
docker compose -f docker-compose.dev.yml logs app

# Restart specific service
docker compose -f docker-compose.dev.yml restart app

# Clean up everything
docker compose -f docker-compose.dev.yml down -v
docker system prune -f
```

#### Database Operations
```bash
# Connect to primary database
docker compose -f docker-compose.dev.yml exec db psql -U postgres -d banking_sim

# Connect to replica database
docker compose -f docker-compose.dev.yml exec db-replica psql -U postgres -d banking_sim

# Backup database
docker compose -f docker-compose.dev.yml exec db pg_dump -U postgres banking_sim > backup.sql
```

#### Application Testing
```bash
# Health check
curl http://localhost:8080/healthz

# API ping
curl http://localhost:8080/api/v1/ping

# Metrics
curl http://localhost:8080/metrics/basic

# Circuit breaker status
curl http://localhost:8080/api/v1/metrics/circuit-breakers

# Circuit breaker testing
curl http://localhost:8080/api/v1/test/circuit-breaker/success
curl http://localhost:8080/api/v1/test/circuit-breaker/failure
curl http://localhost:8080/api/v1/test/circuit-breaker/timeout

# Test circuit breaker opening (run multiple times)
for i in {1..3}; do curl http://localhost:8080/api/v1/test/circuit-breaker/failure; done
```

---

## 🎯 Next Steps

### For Development
1. **Explore the API** using Postman collection
2. **Monitor Performance** via Grafana dashboards
3. **Trace Requests** using Jaeger
4. **Review Logs** for debugging and insights

### For Production
1. **Security Hardening** - Change default secrets, configure TLS
2. **Scaling** - Add more app replicas, configure load balancing
3. **Backup Strategy** - Set up automated database backups
4. **Monitoring Alerts** - Configure alerting rules in Prometheus

### Advanced Features to Explore
- **Circuit Breaker Testing** - Test fault tolerance with `/api/v1/test/circuit-breaker/*` endpoints
- **Event Sourcing** - Review the events table and projector workers
- **Scheduled Transactions** - Create and monitor future transactions
- **Multi-Currency** - Test with different currencies
- **Audit Trails** - Review complete operation history

---

**🎉 Your Go Banking Simulation API is now fully operational!**

This comprehensive system demonstrates enterprise-grade Go development with modern architecture patterns, observability, and production-ready features. Start by registering a user and exploring the financial operations through the Postman collection.
