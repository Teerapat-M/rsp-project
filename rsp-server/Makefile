.PHONY: test test-race run up down migrate-up migrate-down tidy vet sqlc

# Host-side migrate CLI, if installed: https://github.com/golang-migrate/migrate
# Compose already runs the same files via the migrate service (db:5432).
DATABASE_URL ?= postgres://rsp:rsp@localhost:5433/rsp?sslmode=disable
MIGRATE_PATH := db/migrations

# -race needs cgo. `make test` stays plain so Windows without gcc still works.
test:
	go test ./...

test-race:
	go test ./... -race

sqlc:
	sqlc generate

vet:
	go vet ./...

run:
	go run ./cmd/server

up:
	docker compose up -d

down:
	docker compose down -v

migrate-up:
	migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" down

tidy:
	go mod tidy
