COMPOSE := podman compose
COMPOSE_FILE := deploy/docker-compose.yml
GO_CACHE_DIR := $(CURDIR)/.cache/go-build

.PHONY: dev up down logs test lint backend-test frontend-build

dev: up

up:
	$(COMPOSE) -f $(COMPOSE_FILE) up --build

down:
	$(COMPOSE) -f $(COMPOSE_FILE) down

logs:
	$(COMPOSE) -f $(COMPOSE_FILE) logs -f

test: backend-test frontend-build

lint:
	mkdir -p $(GO_CACHE_DIR)
	cd backend && GOCACHE=$(GO_CACHE_DIR) go test ./...
	cd frontend && pnpm --config.verify-deps-before-run=false run lint

backend-test:
	mkdir -p $(GO_CACHE_DIR)
	cd backend && GOCACHE=$(GO_CACHE_DIR) go test ./...

frontend-build:
	cd frontend && pnpm --config.verify-deps-before-run=false run build
