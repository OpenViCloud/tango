VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
CLI_LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

# VPS deploy config — override via env or make args
VPS_HOST ?= root@your-vps-ip
VPS_DIR  ?= /opt/tango

TTL_IMAGE ?= ttl.sh/tango-cloud:24h

.PHONY: dev web-dev test build build-api build-cli build-cli-linux-amd64 build-cli-linux-arm64 build-cli-release build-full sync-static clean-static docker up down infra infra-down push-ttl deploy-test

# Start local dev infra (traefik, postgres, buildkitd, backup-runner) — no app container
infra:
	docker network create tango_net 2>/dev/null || true
	mkdir -p traefik/config letsencrypt
	@if [ ! -f traefik/traefik.yml ]; then \
		printf 'api:\n  dashboard: true\n  insecure: false\nproviders:\n  docker:\n    exposedByDefault: false\n  file:\n    directory: /traefik/config\n    watch: true\nentryPoints:\n  web:\n    address: ":80"\n  websecure:\n    address: ":443"\nping: {}\n' > traefik/traefik.yml; \
		echo "created traefik/traefik.yml"; \
	fi
	@if [ ! -f letsencrypt/acme.json ]; then touch letsencrypt/acme.json && chmod 600 letsencrypt/acme.json; fi
	docker compose -f docker-compose.dev.yml up -d

# Stop local dev infra
infra-down:
	docker compose -f docker-compose.dev.yml down

# Run API locally against dev infra (loads .env.dev)
dev:
	@set -a && . ./.env.dev && set +a && go run ./cmd/api

# Run frontend dev server
web-dev:
	pnpm -C web dev

# Run backend tests
test:
	go test ./...

# Build API-only server
build-api:
	go build -o bin/api ./cmd/api

# Build API server (default target)
build:
	go build -o bin/api ./cmd/api

# Build CLI binary
build-cli:
	go build -ldflags "$(CLI_LDFLAGS)" -o bin/tango ./cmd/cli

# Build Linux AMD64 CLI binary
build-cli-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(CLI_LDFLAGS)" -o bin/tango-linux-amd64 ./cmd/cli

# Build Linux ARM64 CLI binary
build-cli-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags "$(CLI_LDFLAGS)" -o bin/tango-linux-arm64 ./cmd/cli

# Build Linux release CLI binaries
build-cli-release: build-cli-linux-amd64 build-cli-linux-arm64

# Sync frontend build output into Go embed directory
sync-static:
	pnpm -C web build
	rm -rf cmd/api/static/*
	cp -R web/dist/* cmd/api/static/

# Remove embedded frontend assets but keep the folder in git
clean-static:
	rm -rf cmd/api/static/*
	touch cmd/api/static/.gitkeep

# Build full server with embedded frontend
build-full: sync-static
	go build -o bin/api ./cmd/api

# Build Docker image
docker:
	docker build -t tango-cloud .

# Start local stack
up:
	docker compose up -d

# Stop local stack
down:
	docker compose down

# Build multi-arch image and push to ttl.sh (free, no login, expires after 24h)
push-ttl:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(TTL_IMAGE) \
		--push .

# Build, push to ttl.sh, then SSH to VPS and restart app with new image
# Usage: make deploy-test VPS_HOST=root@1.2.3.4 VPS_DIR=/opt/tango
deploy-test: push-ttl deploy-only

# SSH to VPS and restart app — skips build, assumes image already pushed
deploy-only:
	ssh $(VPS_HOST) "mkdir -p $(VPS_DIR)"
	scp docker-compose.test.yml $(VPS_HOST):$(VPS_DIR)/docker-compose.test.yml
	ssh $(VPS_HOST) "cd $(VPS_DIR) && docker network create tango_net 2>/dev/null || true && docker compose -f docker-compose.test.yml pull app && docker compose -f docker-compose.test.yml up -d --no-deps app"
	@echo ""
	@echo "Deployed $(TTL_IMAGE) to $(VPS_HOST):$(VPS_DIR)"
	@echo "Note: image expires in 24h — for production use 'make docker' + push to registry"
