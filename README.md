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
curl -s http://localhost:8080/me/active-room -H 'X-Player-ID: alice'
# { "room": { ... } } or { "room": null } — used so a new tab can restore /play
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
| `make migrate-down` | Roll back **one** migration (no confirmation prompt — Windows cmd.exe-safe). Repeat to go further. |
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

## Deploy

Hosted target: **Go API on Render** (Docker web service, WebSockets), **React on Vercel**, **Postgres on Supabase**. Local/prod-compose images remain: [`rsp-server/Dockerfile`](rsp-server/Dockerfile) and [`rsp-client/Dockerfile`](rsp-client/Dockerfile). No real secrets belong in git.

### Hosted runbook (operator)

1. **Supabase** — New project. Database → Connection string.
   - **API (`DATABASE_URL`)**: Session or transaction **pooler** URI (port **6543**), add `sslmode=require` if missing. Example placeholder: `postgres://postgres.<ref>:<PASSWORD>@aws-0-<region>.pooler.supabase.com:6543/postgres?sslmode=require`
   - **Migrations (`DATABASE_URL_MIGRATE`)**: **Direct** URI (port **5432**, `db.<ref>.supabase.co`). golang-migrate uses session advisory locks; the transaction pooler often fails `migrate up`. Example: `postgres://postgres:<PASSWORD>@db.<ref>.supabase.co:5432/postgres?sslmode=require`
   - If you only have one URI, use the **direct** string for both (fine at this scale).
2. **Render** — New → Blueprint, this repo, [`render.yaml`](render.yaml). Service `rsp-api`, Docker, health `GET /health`. Free web services **sleep after idle**; the first request can take ~30–60s. The client shows “Waking the server…” and retries. Rooms and matches live in API memory: a sleep **drops every in-flight game**.
   - Set dashboard env (blueprint `sync: false`): `DATABASE_URL`, `DATABASE_URL_MIGRATE`, `CORS_ORIGINS` (your Vercel origin, comma-separated), `PUBLIC_BASE_URL` (same origin), `EMAIL_PROVIDER` + mail keys.
   - `JWT_SECRET` is `generateValue` (Render mints it). `RUN_MIGRATIONS=1` (default) runs `migrate up` in the container entrypoint **before** the API listens. To skip: `RUN_MIGRATIONS=0` and run `make migrate-up` against `DATABASE_URL_MIGRATE` yourself.
   - Render injects `PORT`; the process listens on `:$PORT` when `HTTP_ADDR` is empty. `TRUSTED_PROXIES=0.0.0.0/0` so rate limits key on the browser behind Render.
   - Copy the `*.onrender.com` URL (REST is at the root, not `/api`).
3. **Vercel** — Import the repo, **Root Directory `rsp-client`**. [`rsp-client/vercel.json`](rsp-client/vercel.json) rewrites SPA routes to `index.html`.
   - Build env: `VITE_API_BASE=https://<rsp-api>.onrender.com` (no trailing slash, no `/api` prefix), `VITE_WS_URL=wss://<rsp-api>.onrender.com/ws`, optional `VITE_GOOGLE_CLIENT_ID`.
   - Production origin e.g. `https://<app>.vercel.app` must be listed in Render `CORS_ORIGINS` (REST CORS **and** WebSocket `CheckOrigin`).
4. Confirm: `GET https://<rsp-api>.onrender.com/health` → `{"status":"ok","phase":6}`. `GET /ready` pings Postgres. Open the Vercel URL, wait out a cold start if needed, create a room.

Manual migrate (Windows cmd.exe-safe, from `rsp-server/`):

```bash
make migrate-up
```

Uses `DATABASE_URL` (override the DSN on the make command). Repeat `make migrate-down` to roll back one version at a time.

### Production Compose (optional, one box)

```bash
cp .env.example .env   # repo root — fill every required value
docker compose -f docker-compose.prod.yml up --build
# GET http://localhost/health  (via nginx → /api/health)
```

Compose interpolation refuses to start if `POSTGRES_PASSWORD`, `JWT_SECRET`, `CORS_ORIGINS`, `PUBLIC_BASE_URL`, or `EMAIL_PROVIDER` is missing. The API process also refuses `APP_ENV=production` with a missing/dev `JWT_SECRET`, `EMAIL_PROVIDER=log`, empty `CORS_ORIGINS`, or empty `DATABASE_URL`.

Liveness is `GET /health` (process up). Readiness is `GET /ready` (Postgres ping) — that is the Compose healthcheck.

### Environment variables

Who: **operator** (you) unless noted. Never commit `.env`.

| Variable | Service | Required? | Example / placeholder | Purpose |
| --- | --- | --- | --- | --- |
| `APP_ENV` | Render (API); local | prod: yes | `production` / `dev` | Enables prod boot checks; hides `/dev/guest` |
| `PORT` | Render (injected) | Render sets it | `10000` | Listen port when `HTTP_ADDR` is empty |
| `HTTP_ADDR` | local / Compose | no | `:8080` | Listen address; wins over `PORT` |
| `RUN_MIGRATIONS` | Render (API image) | no (default `1`) | `1` / `0` | Entrypoint `migrate up` before listen |
| `DATABASE_URL` | Render, `make migrate-up`, local | prod: yes | `postgres://postgres.<ref>:<PASSWORD>@aws-0-<region>.pooler.supabase.com:6543/postgres?sslmode=require` | API Postgres (pooler OK). Local: `postgres://rsp:rsp@localhost:5433/rsp?sslmode=disable` |
| `DATABASE_URL_MIGRATE` | Render | recommended on Supabase | `postgres://postgres:<PASSWORD>@db.<ref>.supabase.co:5432/postgres?sslmode=require` | Direct DSN for golang-migrate; falls back to `DATABASE_URL` |
| `POSTGRES_USER` | local / Compose | no | `rsp` | Compose database user |
| `POSTGRES_PASSWORD` | Compose | **prod compose yes** | *(set in `.env`, never git)* | Compose database password |
| `POSTGRES_DB` | local / Compose | no | `rsp` | Compose database name |
| `POSTGRES_PORT` | Dev compose | no | `5433` | Host port |
| `POSTGRES_SSLMODE` | Prod compose | no | `disable` / `require` | Compose DSN sslmode (private net: disable) |
| `JWT_SECRET` | Render (generated) / Compose | prod: yes | *(Render generateValue or `.env`)* | Signs access JWTs; not the documented dev default |
| `ACCESS_TOKEN_TTL` | Render / local | no | `15m` | Access token lifetime |
| `REFRESH_TOKEN_TTL` | Render / local | no | `168h` | Refresh lifetime |
| `REFRESH_TOKEN_TTL_REMEMBER` | Render / local | no | `720h` | Refresh with Remember me |
| `PASSWORD_RESET_TTL` | Render / local | no | `1h` | Reset-link lifetime |
| `BCRYPT_COST` | Render / local | no | `12` | Password hash cost |
| `PUBLIC_BASE_URL` | Render / Compose | prod compose yes | `https://<app>.vercel.app` | Origin in password-reset emails |
| `EMAIL_PROVIDER` | Render / Compose | prod: yes | `resend` / `smtp` | `log` forbidden in production |
| `EMAIL_FROM` | Render / Compose | at send time | `RSP <noreply@example.com>` | From header |
| `RESEND_API_KEY` | Render | if resend | `re_xxxxxxxx` | Resend |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USER` `SMTP_PASSWORD` | Render | if smtp | `smtp.example.com` / `587` | SMTP |
| `GOOGLE_CLIENT_ID` | Render | no | Google web client id | Unset disables `POST /auth/google` |
| `CORS_ORIGINS` | Render / Compose | prod: yes | `https://<app>.vercel.app` | REST CORS + WebSocket origin allowlist (comma-separated) |
| `RATE_LIMIT_AUTH_PER_MIN` | Render / local | no | `20` | Unauthenticated auth POSTs per IP per minute |
| `TRUSTED_PROXIES` | Render / Compose | Render: set | `0.0.0.0/0` (Render); Compose private CIDRs | gin `X-Forwarded-For` / ClientIP |
| `DISCONNECT_GRACE` | Render / local | no | `20s` | Seat hold after WS drop |
| `HTTP_PORT` | Prod compose | no | `80` | Host port for nginx |
| `VITE_API_BASE` | Vercel (build) / local | Vercel: yes | `https://<rsp-api>.onrender.com` | REST origin. Local default `/api` (Vite proxy) |
| `VITE_WS_URL` | Vercel (build) / local | Vercel: yes | `wss://<rsp-api>.onrender.com/ws` | Browser WebSocket. Local blank → page host `/ws` |
| `VITE_GOOGLE_CLIENT_ID` | Vercel (build) / local | no | same as `GOOGLE_CLIENT_ID` | Shows Continue with Gmail |

Full lists: [`.env.example`](.env.example) (prod compose), [`rsp-server/.env.example`](rsp-server/.env.example) (local `make run`), [`rsp-client/.env.example`](rsp-client/.env.example) (Vite / Vercel). Never commit `.env`.

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
