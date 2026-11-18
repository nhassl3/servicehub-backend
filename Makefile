.PHONY: build run runb test lint mock sqlc migrate-up migrate-down migrate-force clean docker-build postgres opendb dropdb createdb generate-data redis cli-redis minio minio-stop build-consumer run-consumer runb-consumer kafka-docker \
els-docker els-docker-stop els-reindex-build runb-els-reindex clickhouse-docker clickhouse cli-ch backfill-build runb-backfill clickhouse-open clickhouse-createdb \
clickhouse-up clickhouse-down clickhouse-down-all clickhouse-create clickhouse-force build-only runb-only pgdump openig docker-start docker-stop

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

CLICKHOUSE_HOST ?= localhost
CLICKHOUSE_PORT ?= 19000

# Binary
BINARY_NAME=servicehub
BUILD_DIR=./bin
CMD_PATH=./cmd/servicehub
CONSUMER_PATH=./cmd/consumer
ES_REINDEX_PATH=./cmd/es-reindex
BACKFILL_PATH=./cmd/analytics-backfill
ENVIRONMENT=local
# Migrations
MIGRATE_BIN=$(shell which migrate 2>/dev/null || echo "migrate")
POSTGRES_MIGRATE_PATH=internal/repository/postgres/migrations/
CLICKHOUSE_MIGRATE_PATH=internal/repository/clickhouse/migrations/
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)
CLICKHOUSE_URL=clickhouse://$(CLICKHOUSE_HOST):$(CLICKHOUSE_PORT)?username=$(CLICKHOUSE_USER)&password=$(CLICKHOUSE_PASSWORD)&database=$(CLICKHOUSE_DB)&x-multi-statement=true&x-migrations-table-engine=MergeTree

# SQLC
SQLC_BIN=$(shell which sqlc 2>/dev/null || echo "sqlc")

## ─── Build ───────────────────────────────────────────────────────────────────

export GOOS := $(shell go env GOOS)
export GOARCH := $(shell go env GOARCH)
CGO_ENABLED ?= 0
BUILD_TAGS ?= ""

build: build-consumer els-reindex-build backfill-build
	@echo "Building with: GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED)"
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build \
	-ldflags="-w -s" \
	-o $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH) \
	$(CMD_PATH)
	@echo "Successfully built Main API"

build-only:
	@echo "Building with: GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED)"
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build \
	-ldflags="-w -s" \
	-o $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH) \
	$(CMD_PATH)
	@echo "Successfully built Main API"

run:
	@echo "Run only one API server (not consumer and other services)"
	go run $(CMD_PATH)/main.go

runb: runb-els-reindex runb-backfill # some other needed backfills or reindex
	@ENVIRONMENT=$(ENVIRONMENT) ./$(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH) # application

runb-only:
	@ENVIRONMENT=$(ENVIRONMENT) ./$(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH) # only application

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

DOCKER_DB_IT=postgres18

generate-data:
	@docker exec -i $(CONTAINER_NAME) psql -U servicehub -d servicehub < /home/nhassl3/Projects/servicehub/scripts/seed.sql
	@echo "Successfully created data"

createdb:
	@docker exec -it $(DOCKER_DB_IT) createdb --username=$(DB_USER) --owner=$(DB_USER) $(DB_NAME)

dropdb:
	@docker exec -it $(DOCKER_DB_IT) dropdb --username=$(DB_USER) $(DB_NAME)

opendb:
	@docker exec -it $(DOCKER_DB_IT) psql -U $(DB_USER) $(DB_NAME)

pgdump:
	@docker exec -it $(DOCKER_DB_IT) pg_dump -U $(DB_USER) -d $(DB_NAME) -s -F p -E UTF-8

postgres:
	@docker run --name $(DOCKER_DB_IT) -p 5432:5432 -e POSTGRES_USER=$(DB_USER) -e POSTGRES_PASSWORD=$(DB_PASSWORD) -d postgres:18-alpine

## ─── Migrations ──────────────────────────────────────────────────────────────

migrate-up:
	@$(MIGRATE_BIN) -path $(POSTGRES_MIGRATE_PATH) -database "$(DB_URL)" -verbose up

migrate-down:
	@$(MIGRATE_BIN) -path $(POSTGRES_MIGRATE_PATH) -database "$(DB_URL)" -verbose down 1

migrate-down-all:
	@$(MIGRATE_BIN) -path  $(POSTGRES_MIGRATE_PATH) -database "$(DB_URL)" -verbose down

migrate-force:
	@$(MIGRATE_BIN) -path $(POSTGRES_MIGRATE_PATH) -database "$(DB_URL)" force $(V)

migrate-create:
	@$(MIGRATE_BIN) create -ext sql -dir $(POSTGRES_MIGRATE_PATH) -seq $(NAME)

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

DOCKER_REDIS_IT=redis7

redis:
	@docker run -d -p 127.0.0.1:6380:6380 -v ./redis-config:/usr/local/etc/redis --name $(DOCKER_REDIS_IT) redis:7-alpine redis-server /usr/local/etc/redis/redis.conf --aclfile /usr/local/etc/redis/users.acl

cli-redis:
	@redis-cli -h localhost -p 6380 --user $(REDIS_USER) -a $(REDIS_USER_PASSWORD)

##  ─── MinIO ───────────────────────────────────────────────────────────────────

DOCKER_MINIO_IT=minioRELEASE.2025-09-07T16-13-09Z-cpuv1

minio:
	@docker run -d --name $(DOCKER_MINIO_IT) \
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

DOCKER_KAFKA_IT=servicehub-kafka-local

build-consumer:
	go build -o $(BUILD_DIR)/$(BINARY_NAME)-consumer-$(GOOS)-$(GOARCH) $(CONSUMER_PATH)
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)-consumer-$(GOOS)-$(GOARCH)
	@echo "Successfully built consumer"

run-consumer:
	@go run $(CONSUMER_PATH)

runb-consumer:
	@ENVIRONMENT=$(ENVIRONMENT) ./$(BUILD_DIR)/$(BINARY_NAME)-consumer-$(GOOS)-$(GOARCH)

kafka-docker:
	@docker run -d --name $(DOCKER_KAFKA_IT) -p 9092:9092 apache/kafka:latest

##  ─── GitHub ───────────────────────────────────────────────────────────────────
push:
	@eval "$(ssh-agent -s)"
	@ssh-add servicehub_backend_sshkey

get-contracts:
	@go get -u github.com/nhassl3/servicehub-contracts@$(VER)

##  ─── Elasticsearch (FTS) ──────────────────────────────────────────────────────

DOCKER_ELASTICSEARCH_IT=servicehub-elasticsearch-local

els-docker:
	@docker run -d --name $(DOCKER_ELASTICSEARCH_IT) \
	-p 9200:9200 \
	-e "discovery.type=single-node" \
	-e "xpack.security.enabled=false" \
	-e "ES_JAVA_OPTS=-Xms512m -Xmx512m" \
	elasticsearch:9.3.8
	@echo "Elasticsearch started on http://localhost:9200"

els-docker-stop:
	@docker stop servicehub-elasticsearch-local && docker rm servicehub-elasticsearch-local
	@echo "Elasticsearch stopped"

els-reindex-build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME)-els-reindex-$(GOOS)-$(GOARCH) $(ES_REINDEX_PATH)
	@chmod +x $(BUILD_DIR)/$(BINARY_NAME)-els-reindex-$(GOOS)-$(GOARCH)
	@echo "Successfully built es-reindex"

runb-els-reindex:
	@ENVIRONMENT=$(ENVIRONMENT) ./$(BUILD_DIR)/$(BINARY_NAME)-els-reindex-$(GOOS)-$(GOARCH)

##  ─── Clickhouse (OLAP) ──────────────────────────────────────────────────────

DOCKER_CLICKHOUSE_IT=servicehub-clickhouse-local

clickhouse-docker:
	@docker run -d \
 	--name $(DOCKER_CLICKHOUSE_IT) \
 	-e CLICKHOUSE_USER=$(CLICKHOUSE_USER) \
 	-e CLICKHOUSE_PASSWORD=$(CLICKHOUSE_PASSWORD) \
 	-e CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT=1 \
 	-p 18123:8123 \
 	-p 19000:9000 \
 	--ulimit nofile=262144:262144 \
 	clickhouse/clickhouse-server

clickhouse-open:
	@docker exec -it servicehub-clickhouse-local \
	clickhouse-client \
	-u $(CLICKHOUSE_USER) \
	--password $(CLICKHOUSE_PASSWORD) \
	--database $(CLICKHOUSE_DB)

clickhouse-createdb:
	@docker exec -it servicehub-clickhouse-local \
	clickhouse-client \
	-u $(CLICKHOUSE_USER) \
	--password $(CLICKHOUSE_PASSWORD) \
	-q "CREATE DATABASE IF NOT EXISTS $(CLICKHOUSE_DB);"

clickhouse: clickhouse-docker

cli-ch:
	@docker exec -it servicehub-clickhouse-local clickhouse-client

backfill-build:
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build \
	-ldflags="-w -s" \
	-o $(BUILD_DIR)/$(BINARY_NAME)-backfill-$(GOOS)-$(GOARCH) \
	$(BACKFILL_PATH)
	@echo "Successfully built backfill for clickhouse"

runb-backfill:
	@ENVIRONMENT=$(ENVIRONMENT) ./$(BUILD_DIR)/$(BINARY_NAME)-backfill-$(GOOS)-$(GOARCH)

##  ─── Clickhouse (OLAP) Migrate ───────────────────────────────────────────────────

clickhouse-up:
	@$(MIGRATE_BIN) -path $(CLICKHOUSE_MIGRATE_PATH) -database "$(CLICKHOUSE_URL)" -verbose up

clickhouse-down:
	@$(MIGRATE_BIN) -path $(CLICKHOUSE_MIGRATE_PATH) -database "$(CLICKHOUSE_URL)" -verbose down 1

clickhouse-down-all:
	@$(MIGRATE_BIN) -path  $(CLICKHOUSE_MIGRATE_PATH) -database "$(CLICKHOUSE_URL)" -verbose down

clickhouse-force:
	@$(MIGRATE_BIN) -path $(CLICKHOUSE_MIGRATE_PATH) -database "$(CLICKHOUSE_URL)" force $(V)

clickhouse-create:
	@$(MIGRATE_BIN) create -ext sql -dir $(CLICKHOUSE_MIGRATE_PATH) -seq $(NAME)

##  ─── OpenIG oAuth v2.0 ───────────────────────────────────────────────────

DOCKER_OPENIG_IT=servicehub-openig-local

openig:
	@docker run -d \
	  --name $(DOCKER_OPENIG_IT) \
	  -p 8079:8080 \
	  -v "./openig-config:/usr/local/openig-config:ro" \
	  -e 'CATALINA_OPTS=-Dopenig.base=/usr/local/openig-config -DOAUTH.CLIENT_ID=$(OAUTH_CLIENT_ID) -DOAUTH.CLIENT_SECRET=$(OAUTH_CLIENT_SECRET) -Dendpoint.app=$(ENDPOINT_APP)' \
	  openidentityplatform/openig:6.2.0

## ─── Docker settings ───────────────────────────────────────────────────

docker-start:
	@docker start $(DOCKER_DB_IT) $(DOCKER_MINIO_IT) $(DOCKER_REDIS_IT) $(DOCKER_CLICKHOUSE_IT) $(DOCKER_ELASTICSEARCH_IT) $(DOCKER_KAFKA_IT) $(DOCKER_OPENIG_IT)

docker-stop:
	@docker stop $$(docker ps -a -q)
