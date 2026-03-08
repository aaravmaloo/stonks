SHELL := /bin/sh

COMPOSE := docker compose -f docker-compose.local.yml

.PHONY: build test fmt run-api run-worker local-migrate local-build local-up local-down local-logs local-ps

build:
	go build ./...

test:
	go test ./...

fmt:
	gofmt -w $$(go list -f '{{.Dir}}' ./...)

run-api:
	go run ./cmd/stanks-api

run-worker:
	go run ./cmd/stanks-worker

local-migrate:
	$(COMPOSE) run --rm migrate

local-build:
	$(COMPOSE) build api worker

local-up:
	$(COMPOSE) up -d api worker

local-down:
	$(COMPOSE) down

local-logs:
	$(COMPOSE) logs -f api worker

local-ps:
	$(COMPOSE) ps
