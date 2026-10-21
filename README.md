# rsp-project

Online Rock-Paper-Scissors. Backend is Go; the React client lives in [`web/`](web/).

**You are here: phase 1 + UI mock** — repo scaffold, Postgres schema, in-memory game rules, and a clickable React UI backed by a mock API. Product spec and the six-phase plan: [SPEC.md](SPEC.md).

## Prerequisites

- Go 1.22+
- Docker with Compose v2 (`docker compose`)

## Run Postgres + migrations

```bash
cp .env.example .env   # optional; Compose uses the same defaults
docker compose up -d
```

That starts PostgreSQL and a one-shot `migrate` container (`golang-migrate`) that applies `db/migrations`. Check:

```bash
docker compose ps
docker compose logs migrate
```

Postgres: `localhost:5432`, user/password/db `rsp` (see `.env.example`).

Stop and wipe the volume:

```bash
docker compose down -v
```

## Tests (no Docker required)

```bash
make test
```

This is the phase 1 gate: table-driven tests around `internal/game` (who beats whom, timeout, first-to-N, hidden moves, stale round numbers).

## API stub

Phase 1 only exposes a health check so the module builds as a server:

```bash
make run
# GET http://localhost:8080/health
```

It does not talk to Postgres yet. Rooms (6-digit passcode), WS, and auth (guest / username+password / Google) come in phases 2–4.

## Makefile

| Target | What |
| --- | --- |
| `make test` | `go test ./...` |
| `make run` | `go run ./cmd/server` |
| `make up` / `make down` | Compose up (detached) / down |
| `make migrate-up` | Run migrations from the host (needs [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI) |

## Frontend (React)

The UI is a separate app in `web/` (Vite + TypeScript + dnd-kit). Backend HTTP/WS is not wired yet; the client uses a mock so every screen works.

```bash
cd web
npm install
npm run dev      # http://localhost:5173
npm run build
```

Details, mock passcodes, and env vars: [web/README.md](web/README.md).

## Layout

```
cmd/server          main()
internal/game       rules engine (this phase)
db/migrations       schema
db/queries          sqlc input (later)
sqlc.yaml           sqlc config (generate in later phases)
web/                React + Vite client
```
