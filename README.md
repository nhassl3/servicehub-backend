<p align="center">
  <a href="https://go.dev"><img src="https://img.shields.io/github/go-mod/go-version/nhassl3/servicehub-backend?style=for-the-badge&logo=go&label=" alt="Go"></a>
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/github/license/nhassl3/servicehub-backend?style=flat-square" alt="License"></a>
  <a href="https://github.com/nhassl3/servicehub-backend/actions"><img src="https://img.shields.io/github/actions/workflow/status/nhassl3/servicehub-backend/go.yaml?branch=main&style=flat-square&label=CI" alt="CI"></a>
  <a href="https://github.com/nhassl3/servicehub-backend/releases"><img src="https://img.shields.io/badge/version-v0.4.0-2C8EBB?style=flat-square" alt="Version v0.4.0"></a>
  <a href="https://grpc.io"><img src="https://img.shields.io/badge/gRPC-gateway-244c5a?style=flat-square" alt="gRPC"></a>
  <a href="https://paseto.io"><img src="https://img.shields.io/badge/auth-PASETO%20v4-2C8EBB?style=flat-square" alt="PASETO v4"></a>
  <a href="https://www.postgresql.org"><img src="https://img.shields.io/badge/PostgreSQL-18-4169E1?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL 18"></a>
  <a href="https://redis.io"><img src="https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis&logoColor=white" alt="Redis 7"></a>
  <a href="https://min.io"><img src="https://img.shields.io/badge/MinIO-Object%20Store-72CBE8?style=flat-square&logo=minio&logoColor=white" alt="MinIO"></a>
  <a href="https://kafka.apache.org"><img src="https://img.shields.io/badge/Kafka-3.9-231F20?style=flat-square&logo=apachekafka&logoColor=white" alt="Kafka 3.9"></a>
  <a href="https://www.elastic.co/elasticsearch"><img src="https://img.shields.io/badge/Elasticsearch-9.3-005571?style=flat-square&logo=elasticsearch&logoColor=white" alt="Elasticsearch 9.3"></a>
  <a href="https://www.elastic.co/kibana"><img src="https://img.shields.io/badge/Kibana-9.3-005571?style=flat-square&logo=kibana&logoColor=white" alt="Kibana 9.3"></a>
  <a href="https://clickhouse.com"><img src="https://img.shields.io/badge/ClickHouse-analytics-16253D?style=flat-square&logo=clickhouse&logoColor=white" alt="ClickHouse"></a>
  <a href="https://sqlc.dev"><img src="https://img.shields.io/badge/sqlc-generated-16253D?style=flat-square" alt="sqlc"></a>
  <a href="https://github.com/golang-migrate/migrate"><img src="https://img.shields.io/badge/migrate-golang--migrate-6479EB?style=flat-square" alt="golang-migrate"></a>
  <a href="https://github.com/nhassl3/servicehub-backend/blob/main/CONTRIBUTING.md"><img src="https://img.shields.io/badge/PRs-welcome-brightgreen?style=flat-square" alt="PRs Welcome"></a>
  <a href="https://claude.ai"><img src="https://img.shields.io/badge/Claude-ready-412991?style=flat-square" alt="Claude-ready"></a>
  <a href="https://goreportcard.com/report/github.com/nhassl3/servicehub-backend"><img src="https://goreportcard.com/badge/github.com/nhassl3/servicehub-backend?style=flat-square" alt="Go Report Card"></a>
  <a href="https://hub.docker.com/r/nhassl3/servicehub-backend"><img src="https://img.shields.io/badge/Docker-ready-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker"></a>
</p>

# ServiceHub Backend

> Production-ready marketplace backend built with Go, gRPC, and PostgreSQL — Clean Architecture, PASETO auth, sqlc, and full CI.

ServiceHub is a modern service marketplace platform inspired by the architecture of large-scale e-commerce systems. This repository contains the backend service, written in Go following Clean Architecture principles.

### Changelog Highlights

- **v0.1.0** — Initial release: gRPC + REST gateway, PostgreSQL, PASETO auth, sqlc
- **v0.2.0** — Added Kafka for stream processing (async notifications, event-driven workflows)
- **v0.3.0** — Added Elasticsearch + Kibana for full-text search and observability
- **v0.4.0** — Added ClickHouse for OLAP analytics (event ingestion + materialized aggregates)

---

## Tech Stack

| Layer | Technology |
|:---:|:-------:|
| Language | Go 1.26 |
| Transport | gRPC + REST gateway (grpc-gateway) |
| Database | PostgreSQL 18 (alpine) |
| Cache | Redis 7 |
| Object Storage | MinIO |
| Search | Elasticsearch 9.3.8 |
| Analytics | ClickHouse |
| Stream Processing | Kafka:latest |
| Auth | PASETO v4 / JWT |
| Password hashing | argon2id |
| Query generation | sqlc |
| Migrations | golang-migrate |
| Logging | Uber Zap |
| Config | Viper (YAML + .env) |
| Testing | testify + gomock |
| Container | Docker (multi-stage, alpine) |

---

## Project Structure

```
servicehub-backend/
├── cmd/servicehub/         # Entry point
├── config/                 # YAML configs (local, prod)
├── internal/
│   ├── app/                # Application bootstrap & graceful shutdown
│   ├── config/             # Config loader
│   ├── db/                 # sqlc-generated database layer
│   ├── domain/             # Domain models & interfaces
│   ├── repository/
│   │   ├── postgres/       # PostgreSQL implementations
│   │   └── mock/           # gomock mocks
│   ├── service/            # Business logic + unit tests
│   └── transport/grpc/
│       ├── interceptors/   # Auth, logging, recovery, validation
│       └── *_handler.go    # gRPC handlers
├── migrations/             # SQL migration files (up/down)
├── pkg/
│   ├── auth/               # PASETO / JWT token manager
│   ├── hash/               # argon2id password hasher
│   ├── logger/             # Zap logger factory
│   ├── postgres/           # Connection pool
│   └── validator/          # protovalidate integration
├── Dockerfile              # Multi-stage Docker build
├── Makefile                # Developer shortcuts
└── start.sh                # Container entrypoint (migrate → run)
```

---

## Domain

The platform covers a full marketplace workflow:

- **Users** — registration, authentication (PASETO), profile
- **Sellers** — seller accounts linked to users
- **Products** — catalogue with categories
- **Categories** — product taxonomy
- **Orders / Order Items** — order lifecycle management
- **Cart / Cart Items** — shopping cart
- **Reviews** — product reviews
- **Wishlists** — saved products
- **Balances / Transactions** — internal wallet system

---

## Getting Started

### Prerequisites

- Docker & Docker Compose
- Go 1.25+
- `make`
- [`migrate`](https://github.com/golang-migrate/migrate) CLI (for local development)

### Environment

Copy the example and fill in secrets:

```bash
cp .env.example .env
```

Required variables:

```env
DB_USER=servicehub
DB_PASSWORD=<strong-password>
DB_NAME=servicehub
PASETO_KEY=<32-byte-hex-key>
```

### Run with Docker Compose

From the repository root (one level up):

```bash
docker compose up --build
```

Services exposed:

| Service | Port |
|---|---|
| gRPC | 9090 |
| HTTP gateway | 8080 |
| PostgreSQL | 5432 |

Migrations run automatically on container startup.

### Run locally

```bash
# Start a local Postgres instance
make postgres

# Apply migrations
make migrate-up

# Run the server
make run
```

---

## Makefile Reference

```bash
make build          # Compile binary for current OS/ARCH
make run            # go run ./cmd/servicehub
make test           # Run all tests with race detector + coverage
make test-verbose   # Verbose test output
make cover          # Open coverage report in browser
make lint           # golangci-lint
make mock           # Regenerate gomock mocks
make sqlc           # Regenerate sqlc database layer
make migrate-up     # Apply all pending migrations
make migrate-down   # Roll back one migration
make clean          # Remove build artifacts
```

---

## Architecture

Clean Architecture with strict layer separation:

```
Transport (gRPC) → Service (business logic) → Repository (data access) → DB
```

- **Domain** layer defines interfaces — no framework dependencies
- **Repository** layer is swappable; mocks are generated automatically
- **Service** layer contains all business rules and is fully unit-tested
- **Transport** layer handles serialization, auth, and validation via interceptors

### gRPC Interceptors

| Interceptor | Purpose |
|---|---|
| `auth` | Validates PASETO token, injects claims into context |
| `logging` | Structured request/response logging with Zap |
| `recovery` | Panic recovery with stack trace logging |
| `validation` | Proto field validation via `protovalidate` |

---

## Testing

```bash
make test
```

- Unit tests for all services with gomock
- Token manager tests (PASETO + JWT)
- Password hashing tests
- Race detector enabled by default

---

## Related Repositories

| Repository | Description |
|---|---|
| `servicehub-contracts` | Protobuf definitions shared between backend and frontend |
| `servicehub-frontend` | React + Vite + TypeScript frontend |

---

## License

MIT
