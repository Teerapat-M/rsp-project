# rsp-project

Online Rock-Paper-Scissors. Backend is Go; the React client lives in [`web/`](web/).

**You are here: phase 2** — create/join rooms by 6-digit passcode, an in-memory WebSocket hub, and a temporary player identity. Gameplay countdown and real auth are later. Product spec: [SPEC.md](SPEC.md). WS messages: [docs/ws-protocol.md](docs/ws-protocol.md).

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

## API (phase 2)

```bash
make run
# GET http://localhost:8080/health
```

Identity is **temporary** until phase 4. Send `X-Player-ID: alice` (any string; UUIDs are kept, other values hash to a stable UUID) or mint a guest:

```bash
curl -s http://localhost:8080/dev/guest \
  -H 'Content-Type: application/json' \
  -d '{"displayName":"Alice"}'
# { "accessToken": "<uuid>", "user": { ... } }
```

Use that UUID as `Authorization: Bearer` or as `?token=` on the WebSocket.

### Try two players

```bash
# terminal A — host
curl -s -X POST http://localhost:8080/rooms \
  -H 'Content-Type: application/json' \
  -H 'X-Player-ID: alice' \
  -H 'X-Player-Name: Alice' \
  -d '{"winsToFinish":3,"turnSeconds":15}'
# note id + passcode

# terminal B — join (use a real passcode, not 000000)
curl -s -X POST http://localhost:8080/rooms/join \
  -H 'Content-Type: application/json' \
  -H 'X-Player-ID: bob' \
  -H 'X-Player-Name: Bob' \
  -d '{"passcode":"123456"}'

# wrong code → {"code":"WRONG_PASSCODE", ...}
# third player → {"code":"ROOM_FULL", ...}

curl -s http://localhost:8080/rooms/<id> -H 'X-Player-ID: alice'
```

WebSocket (after joining):

```
ws://localhost:8080/ws?room=<id>&token=alice
```

Send `{"type":"lobby.ready"}` or `{"type":"leave"}`. Presence and `room.updated` are pushed to everyone in the room. Disconnect grace is 20s (`DISCONNECT_GRACE`).

`POST /rooms/:id/start` and move/ready-during-play are **phase 3** (`501 NOT_IMPLEMENTED`).

The React app in `web/` already speaks this REST shape (`/rooms`, `/rooms/join`, error codes `WRONG_PASSCODE` / `ROOM_FULL`). It still defaults to its mock (`VITE_USE_MOCK` not `false`). To point it at Go: mint a guest via `/dev/guest`, store `accessToken` / `user` the way the mock login does, set `VITE_USE_MOCK=false`, and run the Vite proxy (it strips `/api` and forwards to `:8080`).

## Tests

```bash
make test          # go test ./... -race
```

No Docker required: join rules, passcode allocation, the hub, and the WS integration test use an in-memory store.

## Makefile

| Target | What |
| --- | --- |
| `make test` | `go test ./... -race` |
| `make run` | `go run ./cmd/server` |
| `make up` / `make down` | Compose up (detached) / down |
| `make migrate-up` | Run migrations from the host (needs [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI) |
| `make sqlc` | Regenerate `internal/sqlcdb` |

## Frontend (React)

The UI is a separate app in `web/` (Vite + TypeScript + dnd-kit). Mock mode is still the default so every screen is clickable without auth.

```bash
cd web
npm install
npm run dev      # http://localhost:5173
npm run build
```

Details: [web/README.md](web/README.md).

## Layout

```
cmd/server          main()
internal/game       rules engine
internal/ident      temporary player identity
internal/passcode   6-digit codes
internal/rooms      persistence + join rules
internal/hub        live rooms + WS
internal/server     gin routes
internal/sqlcdb     generated
db/migrations       schema
db/queries          sqlc SQL
docs/ws-protocol.md WS types
web/                React + Vite client
```
