# ─── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder
LABEL authors="nhassl3"

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
--mount=type=cache,target=/root/.cache/go-build \
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/servicehub ./cmd/servicehub

RUN --mount=type=cache,target=/go/pkg/mod \
--mount=type=cache,target=/root/.cache/go-build \
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/consumer ./cmd/consumer

RUN --mount=type=cache,target=/go/pkg/mod \
--mount=type=cache,target=/root/.cache/go-build \
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/es-reindex ./cmd/es-reindex

# Install migrate via go install (uses already-downloaded Go modules cache)
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

# ─── Final stage ──────────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binaries
COPY --from=builder /bin/servicehub .
COPY --from=builder /bin/es-reindex .
COPY --from=builder /bin/consumer .
COPY --from=builder /go/bin/migrate ./migrate

# Copy configuration files (DO NOT include .env - it should be provided at runtime)
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/internal/repository/clickhouse/migrations internal/repository/clickhouse/migrations/
COPY --from=builder /app/config/prod.yaml config/prod.yaml
COPY --from=builder /app/config/local.yaml config/local.yaml
COPY --from=builder /app/config/dev.yaml config/dev.yaml
COPY --from=builder /app/.env .

# Copy templates if they exist
COPY --from=builder /app/pkg/mailer/templates/*.html pkg/mailer/templates/
RUN chmod +x /app/migrate /app/servicehub /app/consumer /app/es-reindex

EXPOSE 8082 50051