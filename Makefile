.PHONY: build run runb test lint mock sqlc migrate-up migrate-down migrate-force clean docker-build postgres opendb dropdb createdb generate-data

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
	@./$(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)

## ─── Test ────────────────────────────────────────────────────────────────────

test:
	go test -race -coverprofile=coverage.out ./...

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
	$(SQLC_BIN) generate

## ─── DATABSE CONTROLL ────────────────────────────────────────────────────────

generate-data:
	@docker exec -i servicehub-postgres psql -U servicehub -d servicehub < /home/nhassl3/Projects/servicehub/scripts/seed.sql
	@echo "Successfully created data"

_regenerateData:
	@docker exec -i servicehub-postgres psql -U servicehub -d servicehub \
 	-c "TRUNCATE balance_transactions, balances, wishlists, cart_items, carts, reviews, order_items, orders, products, sellers RESTART IDENTITY CASCADE; DELETE FROM users WHERE username != 'admin';"

regenerate-data: _regenerateData generate-data
	@echo "Data regenerated successfully"

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
