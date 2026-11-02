.DEFAULT_GOAL := help
-include .env
export
PRODUCTION_COMPOSE = env -u APP_ENV -u PUBLIC_BASE_URL -u APP_BIND -u APP_PORT -u ADMIN_USERNAME -u ADMIN_PASSWORD -u POSTGRES_PASSWORD -u MOCK_CORS_ORIGINS -u MOCK_CORS_HEADERS docker compose --project-name mocklab-production --env-file .env.production -f compose.production.yaml

.PHONY: help install generate assets build run dev watch-assets watch-templ test check up down production-up production-down clean
help:
	@echo "install  Install pinned dependencies"
	@echo "dev      Build assets/templates and run the Go server"
	@echo "build    Build the standalone Go binary"
	@echo "test     Run editor and Go tests (PostgreSQL integration: TEST_DATABASE_URL=...)"
	@echo "check    Test and vet"
	@echo "up       Start Go + PostgreSQL using Docker Compose"
	@echo "production-up  Start the internal Go + PostgreSQL stack (.env.production required)"
install:
	npm ci
	go mod download
generate:
	go tool templ generate
assets:
	npm run build
build: generate assets
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/mocklab ./cmd/server
run:
	go run ./cmd/server
dev: generate assets run
watch-assets:
	npm run watch
watch-templ:
	go tool templ generate --watch --cmd="go run ./cmd/server"
test:
	npm test
	go test -race ./...
check: test
	go vet ./...
up:
	docker compose up --build -d
down:
	docker compose down
production-up:
	$(PRODUCTION_COMPOSE) up --build -d
production-down:
	$(PRODUCTION_COMPOSE) down
clean:
	rm -rf bin
