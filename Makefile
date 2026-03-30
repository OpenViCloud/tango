.PHONY: dev web-dev test build build-api build-full sync-static clean-static docker up down

# Run API locally
dev:
	go run ./cmd/api

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
