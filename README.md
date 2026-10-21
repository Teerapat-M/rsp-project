# rsp-project

Online Rock-Paper-Scissors. Backend is Go in [`rsp-server/`](rsp-server/); the React client lives in [`rsp-client/`](rsp-client/).

**You are here: phase 6** — Docker images, a production Compose example, and env-based config. Product spec: [SPEC.md](SPEC.md). WS messages: [docs/ws-protocol.md](docs/ws-protocol.md).

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

Postgres from the host: `localhost:5433` (Compose maps `5433:5432` so a native Postgres on 5432 can stay). User/password/db come from `POSTGRES_*` in `.env` (local default `rsp` — see `.env.example`). Inside the Compose network, migrate talks to `db:5432`. The DSN uses `sslmode=disable` for this private Compose network; use `sslmode=require` (or `verify-full`) if the database is reached over the internet.

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

In **dev** (`APP_ENV=dev`, the default), `POST /dev/guest` is an alias and `X-Player-ID: alice` still works for curl. Production requires `JWT_SECRET` (not the documented dev default), `DATABASE_URL`, `CORS_ORIGINS`, and a real mailer (`EMAIL_PROVIDER=resend` or `smtp` — `log` is refused). `/dev/guest` and `X-Player-ID` are ignored. `GET /health` is liveness; `GET /ready` pings Postgres.

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

The React app in `rsp-client/` speaks this REST + WS shape. Vite proxies `/api` and `/ws` to `:8080` (it strips the `/api` prefix).

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

The UI is a separate app in `rsp-client/` (Vite + TypeScript + dnd-kit). It always talks to the Go server:

```bash
cd rsp-client
cp .env.example .env
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

## Deploy (phase 6)

Images: [`rsp-server/Dockerfile`](rsp-server/Dockerfile) (Go API) and [`rsp-client/Dockerfile`](rsp-client/Dockerfile) (nginx SPA). The browser talks to nginx; `/api/` and `/ws` are proxied to the API. Postgres is not published to the host.

```bash
cp .env.example .env   # repo root — fill every required value
docker compose -f docker-compose.prod.yml up --build
# GET http://localhost/health  (via nginx → /api/health)
```

Compose interpolation refuses to start if `POSTGRES_PASSWORD`, `JWT_SECRET`, `CORS_ORIGINS`, `PUBLIC_BASE_URL`, or `EMAIL_PROVIDER` is missing. The API process also refuses `APP_ENV=production` with a missing/dev `JWT_SECRET`, `EMAIL_PROVIDER=log`, empty `CORS_ORIGINS`, or empty `DATABASE_URL`.

Liveness is `GET /health` (process up). Readiness is `GET /ready` (Postgres ping) — that is the Compose healthcheck.

### Environment variables

| Variable | Where | Notes |
| --- | --- | --- |
| `APP_ENV` | API | `dev` (default) or `production` |
| `HTTP_ADDR` | API | Listen address, default `:8080` |
| `DATABASE_URL` | API / `make migrate-up` | Required in production. Local default `postgres://rsp:rsp@localhost:5433/rsp?sslmode=disable` |
| `POSTGRES_USER` | Compose | Default `rsp` |
| `POSTGRES_PASSWORD` | Compose | Local default `rsp`. **Required in prod compose** (no default) |
| `POSTGRES_DB` | Compose | Default `rsp` |
| `POSTGRES_PORT` | Dev compose | Host port, default `5433` |
| `POSTGRES_SSLMODE` | Prod compose | Default `disable` on the Compose network; use `require` / `verify-full` over the internet |
| `JWT_SECRET` | API | Required in production; must not be the documented dev default |
| `ACCESS_TOKEN_TTL` | API | Default `15m` |
| `REFRESH_TOKEN_TTL` | API | Default `168h` |
| `REFRESH_TOKEN_TTL_REMEMBER` | API | Default `720h` |
| `PASSWORD_RESET_TTL` | API | Default `1h` |
| `BCRYPT_COST` | API | Default `12` (raised to 10+ outside dev) |
| `PUBLIC_BASE_URL` | API | Origin used in password-reset links. Required in prod compose |
| `EMAIL_PROVIDER` | API | `log` (dev) / `resend` / `smtp`. `log` is forbidden in production |
| `EMAIL_FROM` | API | Required by resend and smtp at send time |
| `RESEND_API_KEY` | API | When `EMAIL_PROVIDER=resend` |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USER` `SMTP_PASSWORD` | API | When `EMAIL_PROVIDER=smtp` |
| `GOOGLE_CLIENT_ID` | API | Unset disables `POST /auth/google` |
| `CORS_ORIGINS` | API | Comma-separated browser origins. Required in production (REST + WebSocket) |
| `RATE_LIMIT_AUTH_PER_MIN` | API | Unauthenticated auth POSTs per IP per minute (default 20) |
| `DISCONNECT_GRACE` | API | Default `20s` |
| `HTTP_PORT` | Prod compose | Host port for nginx, default `80` |
| `VITE_API_BASE` | Client build | Default `/api` (nginx strips the prefix) |
| `VITE_WS_URL` | Client | Blank uses the page host `/ws` |
| `VITE_GOOGLE_CLIENT_ID` | Client build | Same value as `GOOGLE_CLIENT_ID`; baked into the image |

Full lists: [`.env.example`](.env.example) (prod compose) and [`rsp-server/.env.example`](rsp-server/.env.example) (local `make run`). [`rsp-client/.env.example`](rsp-client/.env.example) is the Vite dev file. Never commit `.env`.

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
