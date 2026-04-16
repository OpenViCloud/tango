VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
CLI_LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

# VPS deploy config — override via env or make args
VPS_HOST ?= root@your-vps-ip
VPS_DIR  ?= /opt/tango

TTL_IMAGE ?= ttl.sh/tango-cloud:24h

.PHONY: dev web-dev test build build-api build-cli build-cli-linux-amd64 build-cli-linux-arm64 build-cli-release build-release build-full sync-static clean-static docker up down infra infra-down push-ttl deploy-test deploy-test-full deploy-only deploy-agent deploy-agent-full vps-bootstrap vps-install

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

# Run API locally against dev infra (loads .env.dev, auto-creates if missing)
dev:
	@if [ ! -f .env.dev ]; then \
		KEY=$$(LC_ALL=C tr -dc 'a-zA-Z0-9' < /dev/urandom | head -c32); \
		printf '# Local dev environment — loaded by `make dev`\n# Do NOT commit this file\n\nPORT=8080\nAPI_BASE_URL=http://localhost:8080\nFRONTEND_BASE_URL=http://localhost:5173\n\nDB_DRIVER=postgres\nDATABASE_URL=postgres://postgres:postgres@localhost:5432/tango?sslmode=disable\n\nDATA_ENCRYPTION_KEY=%s\n\n# Admin account seeded on first start (skip if already exists)\nADMIN_EMAIL=admin@tango.local\nADMIN_PASSWORD=admin123\n\nBUILDKIT_HOST=tcp://localhost:1234\n\nBACKUP_RUNNER_BASE_URL=http://localhost:8081\nBACKUP_RUNNER_TOKEN=\n\nTRAEFIK_CONFIG_DIR=./traefik/config\nTRAEFIK_DOCKER_NETWORK=tango_net\n\nLOG_FORMAT=text\nLOG_OUTPUT=stdout\n' "$$KEY" > .env.dev; \
		echo "created .env.dev (DATA_ENCRYPTION_KEY auto-generated)"; \
	fi
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

# Check if install.dev.sh has been run on VPS — if not, run it first
# Sentinel: /opt/tango/.env (created by install.dev.sh)
# Usage: make vps-bootstrap VPS_HOST=root@1.2.3.4
vps-bootstrap:
	@echo "→ Checking install state on $(VPS_HOST)..."
	@ssh $(VPS_HOST) "test -f $(VPS_DIR)/.env" 2>/dev/null \
		&& echo "  ✓ Already installed — skipping install.dev.sh" \
		|| $(MAKE) vps-install VPS_HOST=$(VPS_HOST) VPS_DIR=$(VPS_DIR) TTL_IMAGE=$(TTL_IMAGE) VPS_ARCH=$(VPS_ARCH)

# Run install.dev.sh on a fresh VPS
# Usage: make vps-install VPS_HOST=root@1.2.3.4
vps-install:
	@echo "→ Fresh VPS detected — running install.dev.sh on $(VPS_HOST)..."
	ssh $(VPS_HOST) "mkdir -p $(VPS_DIR)/bin"
	scp docker-compose.test.yml $(VPS_HOST):$(VPS_DIR)/docker-compose.test.yml
	scp bin/tango-linux-$(VPS_ARCH) $(VPS_HOST):$(VPS_DIR)/bin/tango-linux-$(VPS_ARCH)
	scp install.dev.sh $(VPS_HOST):$(VPS_DIR)/install.dev.sh
	ssh $(VPS_HOST) " \
		chmod +x $(VPS_DIR)/install.dev.sh && \
		TTL_IMAGE=$(TTL_IMAGE) TANGO_DIR=$(VPS_DIR) bash $(VPS_DIR)/install.dev.sh \
	"
	@echo "→ install.dev.sh complete on $(VPS_HOST)"

# Detect VPS arch and pick the right CLI binary to copy
VPS_ARCH ?= amd64   # override with VPS_ARCH=arm64 if needed

# ── Build targets ──────────────────────────────────────────────────────────

# Build everything needed for deployment (image + CLI for both archs)
# Run this ONCE before deploying to multiple VPS
build-release: push-ttl build-cli-linux-amd64 build-cli-linux-arm64
	@echo "✅ Release artifacts ready"
	@echo "   Image : $(TTL_IMAGE)"
	@echo "   CLI   : bin/tango-linux-amd64 bin/tango-linux-arm64"

# ── Deploy targets (NO build — use existing artifacts) ─────────────────────

# Deploy control plane to VPS — assumes build-release already ran
# Usage: make deploy-test VPS_HOST=root@1.2.3.4
deploy-test: vps-bootstrap deploy-only

# Deploy worker agent to VPS — assumes build-release already ran
# Usage: make deploy-agent VPS_HOST=root@<worker-ip>
deploy-agent: vps-bootstrap
	ssh $(VPS_HOST) "command -v wg >/dev/null 2>&1 || (apt-get update && apt-get install -y wireguard-tools)"
	ssh $(VPS_HOST) "rm -f /usr/local/bin/tango"
	scp bin/tango-linux-$(VPS_ARCH) $(VPS_HOST):/usr/local/bin/tango
	ssh $(VPS_HOST) "chmod +x /usr/local/bin/tango"
	@echo ""
	@echo "✅ Agent deployed to $(VPS_HOST)"
	@echo "   CLI : $$(ssh $(VPS_HOST) tango version)"
	@echo ""
	@echo "   Next steps:"
	@echo "   1. On control plane: make token  VPS_HOST=root@<VPS1_IP>"
	@echo "   2. On this node:     ssh $(VPS_HOST) tango node join --server http://<VPS1_IP>:8080 --token <TOKEN>"

# SSH to VPS and restart app — no build, uses existing image + binary
deploy-only:
	ssh $(VPS_HOST) "mkdir -p $(VPS_DIR)"
	scp docker-compose.test.yml $(VPS_HOST):$(VPS_DIR)/docker-compose.test.yml
	ssh $(VPS_HOST) "rm -f /usr/local/bin/tango"
	scp bin/tango-linux-$(VPS_ARCH) $(VPS_HOST):/usr/local/bin/tango
	ssh $(VPS_HOST) "chmod +x /usr/local/bin/tango"
	ssh $(VPS_HOST) " \
		cd $(VPS_DIR) && \
		docker network create tango_net 2>/dev/null || true && \
		mkdir -p ~/.config/tango && \
		printf '%s\n' \
		  '{' \
		  '  \"driver\": \"compose\",' \
		  '  \"check_interval\": \"30s\",' \
		  '  \"max_retries\": 3,' \
		  '  \"retry_backoff\": \"10s\",' \
		  '  \"retry_cooldown\": \"5m\",' \
		  '  \"compose_file\": \"$(VPS_DIR)/docker-compose.test.yml\",' \
		  '  \"project_name\": \"tango\",' \
		  '  \"health_url\": \"http://localhost:8080/api/status\",' \
		  '  \"services\": [\"app\", \"traefik\", \"db\", \"buildkitd\", \"backup-runner\"]' \
		  '}' > ~/.config/tango/daemon.json && \
		docker compose --env-file $(VPS_DIR)/.env -f docker-compose.test.yml pull && \
		docker compose --env-file $(VPS_DIR)/.env -f docker-compose.test.yml up -d \
	"
	@echo ""
	@echo "✅ Deployed $(TTL_IMAGE) → $(VPS_HOST):$(VPS_DIR)"
	@echo "   CLI : $$(ssh $(VPS_HOST) tango version)"

# ── Convenience: build + deploy in one command ─────────────────────────────

# Build everything then deploy control plane
# Usage: make deploy-test-full VPS_HOST=root@1.2.3.4
deploy-test-full: build-release vps-bootstrap deploy-only

# Build CLI then deploy worker agent
# Usage: make deploy-agent-full VPS_HOST=root@<worker-ip>
deploy-agent-full: build-cli-linux-$(VPS_ARCH) deploy-agent
