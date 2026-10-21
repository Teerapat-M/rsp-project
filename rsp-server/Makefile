.PHONY: test test-race run up down migrate-up migrate-down tidy vet sqlc check-migrations

# Host-side migrate CLI, if installed: https://github.com/golang-migrate/migrate
# Compose already runs the same files via the migrate service (db:5432).
#
# Recipes are one command each (no POSIX shell, no pipelines). GNU make on
# Windows (ezwinports) runs them in cmd.exe; `go`/`docker`/`migrate` accept
# the same argv there as under sh.
DATABASE_URL ?= postgres://rsp:rsp@localhost:5433/rsp?sslmode=disable
MIGRATE_PATH := db/migrations

# golang-migrate rejects two files with the same version (e.g. 000003_a and
# 000003_b). Fail before `go test` / `migrate up` so Compose cannot hit that.
check-migrations:
	go test ./db/migrations -count=1 -run UniqueVersion

# -race needs cgo. `make test` stays plain so Windows without gcc still works.
test: check-migrations
	go test ./...

test-race: check-migrations
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

migrate-up: check-migrations
	migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" down

tidy:
	go mod tidy
