SHELL := /usr/bin/env bash -o errexit -o pipefail -o nounset

CONTAINER ?= $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
REGISTRY ?= localhost
VERSION ?= $(shell git describe --tags --always --dirty)

API_PORT ?= 8080

dev: # @HELP run the API and Vite dev server together (Ctrl+C stops both)
dev: web/node_modules
	trap 'kill 0' EXIT; \
	PORT=$(API_PORT) go run ./cmd/api & \
	API_URL=http://localhost:$(API_PORT) npm --prefix web run dev & \
	wait

web/node_modules: web/package-lock.json
	npm --prefix web ci
	touch $@

build: # @HELP build the API binary and the frontend bundle
build: web/node_modules
	go build -o bin/api ./cmd/api
	npm --prefix web run build

test: # @HELP run Go tests
test:
	go test ./...

lint: # @HELP go vet and frontend type-check
lint: web/node_modules
	go vet ./...
	cd web && npx vue-tsc -b

images: # @HELP build container images for the API and frontend
images:
	$(CONTAINER) build -t $(REGISTRY)/resonance-api:$(VERSION) .
	$(CONTAINER) build -t $(REGISTRY)/resonance-web:$(VERSION) web

clean: # @HELP remove build artifacts
clean:
	rm -rf bin web/dist

help: # @HELP print this message
help:
	grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
	    | awk 'BEGIN {FS = ": *# *@HELP"}; { printf "  %-12s %s\n", $$1, $$2 }'

.DEFAULT_GOAL := help
.SILENT: help
.PHONY: dev build test lint images clean help
