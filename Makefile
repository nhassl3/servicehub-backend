.PHONY: build run runb test lint mock sqlc migrate-up migrate-down migrate-force clean docker-build postgres opendb dropdb createdb generate-data redis cli-redis minio minio-stop build-consumer run-consumer runb-consumer kafka-docker

.DEFAULT_GOAL := build

# Load secrets from .env (DB_USER, DB_PASSWORD, DB_NAME, PASETO_KEY)
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Public config defaults (can be overridden by environment)
DB_HOST     ?= localhost
DB_PORT     ?= 5432
DB_SSL_MODE ?= disable

# Binary
BINARY_NAME=servicehub
BUILD_DIR=./bin
CMD_PATH=./cmd/servicehub
CONSUMER_PATH=./cmd/consumer
ENVIRONMENT=local

# Migrations
MIGRATE_BIN=$(shell which migrate 2>/dev/null || echo "migrate")
MIGRATE_PATH=file://migrations
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

# SQLC
SQLC_BIN=$(shell which sqlc 2>/dev/null || echo "sqlc")

## ─── Build ───────────────────────────────────────────────────────────────────

export GOOS := $(shell go env GOOS)
export GOARCH := $(shell go env GOARCH)
CGO_ENABLED ?= 0
BUILD_TAGS ?= ""

build:
	@echo "Building with: GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED)"
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build \
	-ldflags="-w -s" \
	-o $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH) \
	$(CMD_PATH)
	@echo "Successfully built"

run:
	go run $(CMD_PATH)/main.go

runb:
	@ENVIRONMENT=$(ENVIRONMENT) ./$(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)

## ─── Test ────────────────────────────────────────────────────────────────────

test:
	@CGO_ENABLED=1 go test -race -coverprofile=coverage.out ./...

test-verbose:
	go test -race -v ./...

cover:
	go tool cover -html=coverage.out

## ─── Lint ────────────────────────────────────────────────────────────────────

lint:
	golangci-lint run ./...

## ─── Mocks ───────────────────────────────────────────────────────────────────

mock:
	go generate ./internal/domain/...

## ─── SQLC ────────────────────────────────────────────────────────────────────

sqlc:
	@$(SQLC_BIN) generate
	@echo "Successfully built SQL code"

## ─── DATABSE CONTROLL ────────────────────────────────────────────────────────

generate-data:
	@docker exec -i $(CONTAINER_NAME) psql -U servicehub -d servicehub < /home/nhassl3/Projects/servicehub/scripts/seed.sql
	@echo "Successfully created data"

createdb:
	@docker exec -it postgres18 createdb --username=$(DB_USER) --owner=$(DB_USER) $(DB_NAME)

dropdb:
	@docker exec -it postgres18 dropdb --username=$(DB_USER) $(DB_NAME)

opendb:
	@docker exec -it postgres18 psql -U $(DB_USER) $(DB_NAME)

postgres:
	@docker run --name postgres18 -p 5432:5432 -e POSTGRES_USER=$(DB_USER) -e POSTGRES_PASSWORD=$(DB_PASSWORD) -d postgres:18-alpine

## ─── Migrations ──────────────────────────────────────────────────────────────

migrate-up:
	@$(MIGRATE_BIN) -path migrations -database "$(DB_URL)" -verbose up

migrate-down:
	@$(MIGRATE_BIN) -path migrations -database "$(DB_URL)" -verbose down 1

migrate-down-all:
	@$(MIGRATE_BIN) -path migrations -database "$(DB_URL)" -verbose down

migrate-force:
	@$(MIGRATE_BIN) -path migrations -database "$(DB_URL)" force $(V)

migrate-create:
	@$(MIGRATE_BIN) create -ext sql -dir migrations -seq $(NAME)

## ─── Docker ──────────────────────────────────────────────────────────────────

docker-build:
	@docker build -t servicehub-backend .

## ─── Clean ───────────────────────────────────────────────────────────────────

clean:
	@rm -rf $(BUILD_DIR) coverage.out

## ─── Go tools ────────────────────────────────────────────────────────────────

tidy:
	@go mod tidy

vet:
	@go vet ./...

##  ─── Redis ───────────────────────────────────────────────────────────────────
redis:
	@docker run -d -p 127.0.0.1:6380:6380 -v ./redis-config:/usr/local/etc/redis --name redis7 redis:7-alpine redis-server /usr/local/etc/redis/redis.conf --aclfile /usr/local/etc/redis/users.acl

cli-redis:
	@redis-cli -h localhost -p 6380 --user $(REDIS_USER) -a $(REDIS_USER_PASSWORD)

##  ─── MinIO ───────────────────────────────────────────────────────────────────
minio:
	@docker run -d --name minioRELEASE.2025-09-07T16-13-09Z-cpuv1 \
		-p 9000:9000 -p 9001:9001 \
		-e MINIO_ROOT_USER=$(MINIO_ACCESS_KEY) \
		-e MINIO_ROOT_PASSWORD=$(MINIO_SECRET_KEY) \
		-v servicehub-minio-data:/data \
		minio/minio:latest server /data --console-address ":9001"
	@echo "MinIO started: API http://localhost:9000 | Console http://localhost:9001"

minio-stop:
	@docker stop servicehub-minio && docker rm servicehub-minio
	@echo "MinIO stopped"

##  ─── Kafka ───────────────────────────────────────────────────────────────────
build-consumer:
	@go build -o $(BUILD_DIR)/$(BINARY_NAME)-consumer-$(GOOS)-$(GOARCH) $(CONSUMER_PATH)
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)-consumer-$(GOOS)-$(GOARCH)
	@echo "Successful built consumer"

run-consumer:
	@go run $(CONSUMER_PATH)

runb-consumer:
	@ENVIRONMENT=$(ENVIRONMENT)  ./$(BUILD_DIR)/$(BINARY_NAME)-consumer-$(GOOS)-$(GOARCH)

kafka-docker:
	@docker run -d --name servicehub-kafka-local -p 9092:9092 apache/kafka:latest
