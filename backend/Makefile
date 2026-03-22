.PHONY: run build tidy db-up db-down up down logs lint test

# ── Dev ───────────────────────────────────────────────────────────────────────
run:
	go run ./cmd/server

build:
	go build -o bin/tachigo ./cmd/server

tidy:
	go mod tidy

# ── Docker DB (for local dev without a full docker-compose) ───────────────────
db-up:
	docker run -d \
		--name tachigo-postgres \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=tachigo \
		-p 5432:5432 \
		postgres:16-alpine

db-down:
	docker stop tachigo-postgres && docker rm tachigo-postgres

# ── Docker Compose (full stack, run from repo root) ───────────────────────────
up:
	docker compose -f ../docker-compose.yml up --build -d

down:
	docker compose -f ../docker-compose.yml down

logs:
	docker compose -f ../docker-compose.yml logs -f app

# ── Lint / test ───────────────────────────────────────────────────────────────
lint:
	golangci-lint run ./...

test:
	go test ./... -v
