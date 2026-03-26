.PHONY: dev build build-api build-full sync-static clean-static build-cli docker migration

# Chạy dev local
dev:
	go run ./cmd/api

# Build API-only server
build-api:
	go build -o bin/api ./cmd/api

# Build API server (default: API-only)
build:
	go build -o bin/api ./cmd/api

# Sync frontend build output into Go embed directory
sync-static:
	cd web && pnpm build
	rm -rf cmd/api/static/*
	cp -R web/dist/* cmd/api/static/

# Remove embedded frontend assets but keep the folder in git
clean-static:
	rm -rf cmd/api/static/*
	touch cmd/api/static/.gitkeep

# Build full server with embedded frontend
build-full: sync-static
	go build -o bin/api ./cmd/api

# Build CLI cho mọi OS
build-cli:
	GOOS=linux   GOARCH=amd64 go build -o bin/demo_linux_amd64   ./cmd/cli
	GOOS=linux   GOARCH=arm64 go build -o bin/demo_linux_arm64   ./cmd/cli
	GOOS=darwin  GOARCH=amd64 go build -o bin/demo_darwin_amd64  ./cmd/cli
	GOOS=darwin  GOARCH=arm64 go build -o bin/demo_darwin_arm64  ./cmd/cli
	GOOS=windows GOARCH=amd64 go build -o bin/demo_windows_amd64.exe ./cmd/cli
	@echo "✅ CLI binaries built in ./bin/"

# Build Docker image
docker:
	docker build -t tango .

# Chạy với docker-compose
up:
	docker compose up -d

down:
	docker compose down

migration:
	@test -n "$(name)" || (echo 'Usage: make migration name=add_address_to_users' && exit 1)
	migrate create -ext sql -dir migrations -seq $(name)
