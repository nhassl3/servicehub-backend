# ─── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder
LABEL authors="nhassl3"

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/consumer ./cmd/consumer

# ─── Final stage ──────────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/consumer .
COPY --from=builder /app/config/prod.yaml config/prod.yaml
COPY --from=builder /app/config/local.yaml config/local.yaml
COPY --from=builder /app/.env .

ENV ENVIRONMENT=dev

EXPOSE 9092

CMD ["/app/consumer"]
