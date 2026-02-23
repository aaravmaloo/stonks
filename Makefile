SHELL := /bin/sh

.PHONY: build test fmt run-api run-worker

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
