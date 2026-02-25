# ServiceHub Backend

> Production-ready marketplace backend built with Go, gRPC, and PostgreSQL.

ServiceHub is a modern service marketplace platform inspired by the architecture of large-scale e-commerce systems. This repository contains the backend service, written in Go following Clean Architecture principles.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| Transport | gRPC + REST gateway (grpc-gateway) |
| Database | PostgreSQL 18 (alpine) |
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
