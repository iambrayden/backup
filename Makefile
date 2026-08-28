SHELL := /bin/bash
.DEFAULT_GOAL := help

MASTER_KEY ?= $(shell openssl rand -base64 32)
JWT_SECRET  ?= $(shell openssl rand -base64 32)

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: ## Copy .env.example to .env and generate secrets
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		MK=$$(openssl rand -base64 32); JK=$$(openssl rand -base64 32); \
		sed -i "s|REPLACE_WITH_BASE64_32_BYTES|$$MK|1" .env; \
		sed -i "s|REPLACE_WITH_BASE64_32_BYTES|$$JK|1" .env; \
		echo "✅ .env created with generated secrets"; \
	else \
		echo "⚠️  .env already exists"; \
	fi

.PHONY: up
up: ## Start all services
	docker compose up --build -d

.PHONY: down
down: ## Stop all services
	docker compose down

.PHONY: logs
logs: ## Tail all logs
	docker compose logs -f

.PHONY: test-pkg
test-pkg: ## Run shared package tests
	cd pkg && go test ./... -v -race -count=1

.PHONY: test
test: test-pkg ## Run all tests

.PHONY: build-server
build-server: ## Build the API server binary
	cd server && go build -o ../bin/server ./cmd/server

.PHONY: build-agent
build-agent: ## Build the agent binary for current OS/arch
	cd agent && go build -o ../bin/backup-agent ./cmd/agent

.PHONY: build-agent-linux-amd64
build-agent-linux-amd64: ## Cross-compile agent for Linux amd64
	cd agent && GOOS=linux GOARCH=amd64 go build -o ../bin/backup-agent-linux-amd64 ./cmd/agent

.PHONY: build-agent-windows-amd64
build-agent-windows-amd64: ## Cross-compile agent for Windows amd64
	cd agent && GOOS=windows GOARCH=amd64 go build -o ../bin/backup-agent-windows-amd64.exe ./cmd/agent

.PHONY: build-agent-darwin-arm64
build-agent-darwin-arm64: ## Cross-compile agent for macOS Apple Silicon
	cd agent && GOOS=darwin GOARCH=arm64 go build -o ../bin/backup-agent-darwin-arm64 ./cmd/agent

.PHONY: tidy
tidy: ## Run go mod tidy across all modules
	cd pkg && go mod tidy
	cd server && go mod tidy
	cd agent && go mod tidy
	go work sync

.PHONY: web-dev
web-dev: ## Start Next.js dev server
	cd web && npm run dev

.PHONY: web-install
web-install: ## Install web dependencies
	cd web && npm install --legacy-peer-deps

.PHONY: demo-backup
demo-backup: ## Run agent backup against local stack (requires .env)
	@set -a && source .env && set +a && \
	./bin/backup-agent \
		--config /tmp/demo-agent-config.yaml \
		--backup-now
