#!/bin/sh
set -e

DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST:-localhost}:${DB_PORT:-5432}/${DB_NAME}?sslmode=disable"

echo "Running migrations..."
/app/migrate -database "$DB_URL" -path /app/migrations -verbose up

echo "Migration completed. Starting application..."
exec "$@"