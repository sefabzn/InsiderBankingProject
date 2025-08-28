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
- **Circuit Breakers** - Fault tolerance
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
- **Event Sourcing** - Review the events table and projector workers
- **Scheduled Transactions** - Create and monitor future transactions
- **Multi-Currency** - Test with different currencies
- **Audit Trails** - Review complete operation history

---

**🎉 Your Go Banking Simulation API is now fully operational!**

This comprehensive system demonstrates enterprise-grade Go development with modern architecture patterns, observability, and production-ready features. Start by registering a user and exploring the financial operations through the Postman collection.
