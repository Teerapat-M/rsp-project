# rsp-project

Online Rock-Paper-Scissors. Backend is Go in [`rsp-server/`](rsp-server/); the React client lives in [`rsp-client/`](rsp-client/).

**You are here: phase 5** — stats/history plus the React client talking to the real Go API. Product spec: [SPEC.md](SPEC.md). WS messages: [docs/ws-protocol.md](docs/ws-protocol.md).

## Prerequisites

- Go 1.22+
- Docker with Compose v2 (`docker compose`)

## Run Postgres + migrations

From `rsp-server/`:

```bash
cp .env.example .env   # optional; Compose uses the same defaults
docker compose up -d
```

That starts PostgreSQL and a one-shot `migrate` container (`golang-migrate`) that applies `db/migrations`. Check:

```bash
docker compose ps
docker compose logs migrate
```

Postgres from the host: `localhost:5433` (Compose maps `5433:5432` so a native Postgres on 5432 can stay). User/password/db `rsp` (see `.env.example`). Inside the Compose network, migrate talks to `db:5432`.

Stop and wipe the volume:

```bash
docker compose down -v
```

## API (phase 2)

```bash
cd rsp-server
make run
# GET http://localhost:8080/health
```

Auth is JWT. Register / login / guest / Google mint `{ accessToken, refreshToken, user }`. Send `Authorization: Bearer <access>` on REST and `?token=<access>` on the WebSocket.

```bash
curl -s http://localhost:8080/auth/guest \
  -H 'Content-Type: application/json' \
  -d '{"displayName":"Alice","avatar":"rock","color":"#9DB89A"}'
# { "accessToken": "<jwt>", "refreshToken": "<opaque>", "user": { ... } }
```

In **dev** (`APP_ENV=dev`, the default), `POST /dev/guest` is an alias and `X-Player-ID: alice` still works for curl. Production requires `JWT_SECRET` and ignores those shortcuts.

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

### Play a match (phase 3)

After two players have joined and opened `/ws`:

```bash
# host starts (only the host, and only with 2 players)
curl -s -X POST http://localhost:8080/rooms/<id>/start -H 'X-Player-ID: alice'

# on each WebSocket:
# {"type":"submit","move":"rock"}
# {"type":"ready"}
```

The server hides each move until both are Ready or `turnSeconds` elapses. Anyone without a card when the timer fires gets a random rock/paper/scissors (`auto_picked`). First to `winsToFinish` ends the match. Rounds and the match are written to Postgres; `user_stats` updates at match end.

Disconnect mid-match pauses the clock (grace, default 20s). Come back in time and play continues; otherwise it is a forfeit. After `match.finished`, send `{"type":"play_again"}` (both players) for a rematch, or `leave` to return to the lobby.

The React app in `rsp-client/` already speaks this REST + WS shape. It still defaults to its mock (`VITE_USE_MOCK` not `false`). Phase 5 wires the real client; until then you can mint a guest via `/auth/guest`, store `accessToken` / `user`, set `VITE_USE_MOCK=false`, and run the Vite proxy (it strips `/api` and forwards to `:8080`).

## Tests

```bash
cd rsp-server
make test          # go test ./...  (no cgo)
make test-race     # go test ./... -race  (needs cgo)
```

No Docker required: join rules, passcode allocation, the hub, and the WS integration test use an in-memory store.

## Makefile

Run these from `rsp-server/`.

| Target | What |
| --- | --- |
| `make test` | `go test ./...` |
| `make test-race` | `go test ./... -race` (needs cgo) |
| `make run` | `go run ./cmd/server` |
| `make up` / `make down` | Compose up (detached) / down |
| `make migrate-up` | Run migrations from the host (needs [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI) |
| `make sqlc` | Regenerate `internal/sqlcdb` |

## Frontend (React)

The UI is a separate app in `rsp-client/` (Vite + TypeScript + dnd-kit). Mock mode is still the default so every screen is clickable without a backend. Point it at Go:

```bash
cd rsp-client
cp .env.example .env
# VITE_USE_MOCK=false
# VITE_GOOGLE_CLIENT_ID=   # optional; hides Gmail unless set
npm install
npm run dev      # http://localhost:5173 — Vite proxies /api and /ws to :8080
npm run build
```

Two-player smoke against live Postgres (server already up):

```bash
cd rsp-server
make smoke
```

Details: [rsp-client/README.md](rsp-client/README.md).

## Layout

```
rsp-server/cmd/server     main()
rsp-server/internal/game  rules engine
rsp-server/internal/auth  JWT, refresh, Google, mail
rsp-server/internal/ident player identity
rsp-server/cmd/smoke      two-player e2e against live HTTP+WS
rsp-server/internal/passcode  6-digit codes
rsp-server/internal/rooms persistence + join rules
rsp-server/internal/hub   live rooms + WS
rsp-server/internal/server gin routes
rsp-server/internal/sqlcdb generated
rsp-server/db/migrations  schema
rsp-server/db/queries     sqlc SQL
docs/ws-protocol.md       WS types
rsp-client/               React + Vite client
```
