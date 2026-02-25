FROM golang:1.25-alpine AS builder
LABEL authors="nhassl3"

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/servicehub ./cmd/servicehub

# Install migrate via go install (uses already-downloaded Go modules cache)
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

# ─── Final stage ──────────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/servicehub .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /go/bin/migrate ./migrate
COPY --from=builder /app/config/prod.yaml config/prod.yaml
COPY --from=builder /app/config/local.yaml config/local.yaml
COPY --from=builder /app/.env .
COPY --from=builder /app/start.sh .

RUN chmod +x /app/start.sh /app/migrate

ENV ENVIRONMENT=prod

EXPOSE 8080 9090

ENTRYPOINT ["/app/start.sh"]
CMD ["/app/servicehub"]
