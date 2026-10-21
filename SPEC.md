# Rock-Paper-Scissors — product spec

Online 1v1 Rock-Paper-Scissors. This is a junior full-stack portfolio project: a small, complete game with auth, rooms, a live match, stats, and a React UI.

Target scale is small (~32 concurrent pairs). That number drives one important architecture choice: **active room and match state lives in the Go process memory**. PostgreSQL stores users, auth links, room metadata, and match/round results after a round or match ends — not every keystroke.

---

## Product features

### Auth

Three ways in. Every path mints a short-lived **access JWT**. The server trusts that JWT (not the client) for who is in a room.

| Path | How | `users` row |
| --- | --- | --- |
| **Username / password** | Register with username, password, optional email. Login with username + password. | `is_guest = false`, unique `username`, `password_hash` |
| **Google OAuth** | Sign in with Google. | `is_guest = false`, `email` from Google, row in `auth_providers` |
| **Guest** | Pick a display name. No username, email, or password. | `is_guest = true` |

**Register** (password path): username, password, optional email. Display name defaults to the username if the UI does not ask for a separate one. Username is unique ignoring case, 3–20 chars `[A-Za-z0-9_]`. Email is unique when present; omitted email means **Forgot password** is unavailable for that account.

**Login:** username + password. (Email is optional, so it is not the login id.)

**Remember me** (login checkbox): still a short access JWT, plus a **refresh token** with a longer `expires_at` (about 30 days vs ~7 days when unchecked). The raw token is shown once; the DB stores only a hash. Logout revokes it. Phase 4 picks the exact TTLs via env.

**Forgot password:** submit the account email. If it matches a password user, insert a one-use `password_reset_tokens` row (hashed, short TTL) and send a link. In **dev**, skip a real mailer: log the reset URL. Always return the same response whether the email exists or not (no account enumeration). Users with no email cannot reset this way.

**Passwords:** hash with **bcrypt** (`golang.org/x/crypto/bcrypt`, cost ≥ 10). Never store or log the plaintext.

**Guest → registered (upgrade):** keep the same `users.id` so stats and match history stay. An authenticated guest sets username + password (+ optional email), then `is_guest = false`. Reject if the username (or email) is taken. This is simpler than merging a guest into some other account.

Google and password can later be linked on the same user (`auth_providers` + `password_hash`). Phase 4 ships the three entry paths, remember-me, reset, and guest upgrade. Linking Google onto an existing password user can wait if time is tight.

### Rooms

- Host creates a room with:
  - `wins_to_finish` — first player to this many **round wins** ends the match (draws do not count).
  - `turn_seconds` — countdown length for each round.
- The server generates a **6-digit passcode** (`000000`–`999999`). Passcodes are unique among rooms that are not `closed` (partial unique index). After close, the code may be reused. Collision → generate again.
- Others join by passcode. A room is two players.

### Gameplay

1. Each player drags one of rock / paper / scissors into a drop slot, then presses **Ready**.
2. The **server is authoritative**. It stores both choices but **does not reveal** a player's move to the opponent until the round resolves.
3. The round resolves when **both are Ready**, or when the **turn timer ends**.
4. No submitted move when the timer ends → the server assigns a **random** rock / paper / scissors (`auto_picked`). The round then resolves with the usual cycle. A submitted-but-not-ready card still counts.
5. Score is round wins. Play continues until one player reaches `wins_to_finish`.

A player may change their card **until** they press Ready. Ready without a card is rejected. After Ready, the move is locked.

The clock is owned by the server process (WebSocket layer), not by the pure rules package. The rules package receives `Timeout(roundNo)` so a late timer from round N cannot resolve round N+1.

### Stats and history

Per user, persisted at round/match end:

- matches played / wins / losses
- how often they played rock / paper / scissors (auto-picked throws count)
- match history (rooms → matches → rounds → moves)

---

## Data model

PostgreSQL. UUIDs for public ids. `rsp-server/db/migrations/000001_init` then `000002_auth_and_passcode` (do not edit applied v1).

| Table | Role |
| --- | --- |
| `users` | Account. Guests: `is_guest`, no username/email/hash. Password users: unique `username` + `password_hash`, email optional. |
| `auth_providers` | Google (and future IdPs): `(provider, provider_user_id)` unique. Guests have no row here. |
| `refresh_tokens` | Hashed refresh tokens; `expires_at` is longer when Remember me was checked. |
| `password_reset_tokens` | Hashed, single-use, expiring. |
| `user_stats` | 1:1 with user, updated when a match ends. |
| `rooms` | Host, 6-digit `passcode`, settings, `waiting` / `playing` / `closed`. |
| `room_players` | Who joined which room. |
| `matches` | One game series in a room. `winner_id` null until someone wins (or stays null if abandoned). |
| `rounds` | `winner_id` null means draw. |
| `round_moves` | `rock` / `paper` / `scissors`, plus `auto_picked` when the timer chose it. |

Adjustments from the original draft (kept small):

- `matches.winner_id` is nullable so an in-progress or abandoned match can be stored.
- Username/password, refresh tokens, and reset tokens added in v2 (Penpot review). Email is no longer required for every non-guest — password users may omit it.
- `rooms.passcode` is exactly six digits (`^[0-9]{6}$`), unique among non-closed rooms.
- Check constraints: `wins_to_finish >= 1`, `turn_seconds >= 1`, guest has no credentials.
- Indexes on FKs used by history queries (`matches.room_id`, `rounds.match_id`, `auth_providers.user_id`).

Active gameplay (hidden moves, ready flags, remaining seconds) is **not** in these tables. It lives on the in-memory match in the Go server.

---

## Architecture (whole project)

```
Browser (React + Vite + dnd-kit, /web)
    │  REST: auth (guest / password / Google), create/join room, stats
    │  WebSocket: match events
    ▼
Go API (gin) + WS hub
    │  in-memory rooms / matches   ← internal/game is the rules engine
    ▼
PostgreSQL   (users, rooms, persisted results)
```

Stack: Go 1.22+, gin, gorilla/websocket, PostgreSQL, sqlc, golang-migrate, Docker Compose.

Compose publishes Postgres on **localhost:5433** (`5433:5432`) so a native Postgres on 5432 can coexist. Host tools (`DATABASE_URL`, `make migrate-up`, `cmd/server` default) use `localhost:5433`. Inside the Compose network the migrate service uses `db:5432`.

**gorilla/websocket** rather than coder/websocket: more examples a junior can cite, same job at this scale.

**sqlc** rather than an ORM: SQL stays visible (migrations + `db/queries`), generated Go is boring and typed.

**In-memory match state** is fine at ~32 pairs. The ceiling is "one process holds all live games." Phase 6 can stay a single VM; Redis/multi-instance is an upgrade path, not a requirement.

Layout:

```
rsp-server/cmd/server     process entry (HTTP)
rsp-server/internal/game  pure rules — no I/O, no clocks
rsp-server/internal/ident TEMPORARY player identity (X-Player-ID / dev guest token)
rsp-server/internal/passcode  6-digit codes
rsp-server/internal/rooms create/join/leave + sqlc store
rsp-server/internal/hub   in-memory live rooms + WebSocket
rsp-server/internal/server gin HTTP
rsp-server/internal/sqlcdb sqlc output
rsp-server/db/migrations  golang-migrate
rsp-server/db/queries     sqlc input
docs/ws-protocol.md       WebSocket message types
rsp-client/               React app (mock API; HTTP client already matches this phase)
```

---

## Six-phase roadmap

| Phase | Ships | Does not ship |
| --- | --- | --- |
| **1 — logic + scaffold** | `SPEC.md`, idiomatic repo, Compose + Postgres + initial schema, `internal/game` + table-driven tests | HTTP game API, WS, auth, UI |
| **2 — rooms / passcode / websocket** (this PR) | Create/join room by passcode, WS connect with a **dev guest token** (JWT in phase 4), presence + ready-lobby + disconnect grace | Real Google/guest login, countdown / moves / resolve |
| **3 — countdown / ready / resolve** | Server timer, submit + ready, hide moves, resolve, `wins_to_finish`, persist round/match rows | Auth, polished UI |
| **4 — auth** | Guest JWT; username/password register + login; Remember me (refresh token); Forgot password (dev: log the link); bcrypt; guest upgrade to a password account; then Google OAuth | Stats page, drag-drop |
| **5 — stats / history + React drag-drop** | `user_stats` + history API, React + Vite in `rsp-client`, dnd-kit cards → drop slot → Ready | Production hosting |
| **6 — deploy** | Dockerfile, Compose or a small VM, env-based config, README for a stranger to run it | Extra game modes |

Each phase is one PR. Do not fold later phases into this one.

---

## Phase 1 acceptance

- [x] Spec and roadmap in `SPEC.md`
- [x] `rsp-server/` (`cmd/`, `internal/`, `go.mod`, `Makefile`, Compose Postgres, migrations, `.env.example`), README
- [x] `internal/game` covers move resolution, timeout auto-pick, round/match state, `wins_to_finish`
- [x] Table-driven unit tests, `go test ./...` passing
- [x] `docker compose up` starts Postgres and applies migrations

---

## Phase 2 acceptance

- [x] `POST /rooms` with `winsToFinish` / `turnSeconds` mints a unique 6-digit passcode, persists `rooms` + host in `room_players`
- [x] `POST /rooms/join` by passcode; `WRONG_PASSCODE` (404) and `ROOM_FULL` (409) match the React client in `rsp-client/`
- [x] `POST /rooms/:id/leave` and `GET /rooms/:id`
- [x] `GET /ws?room=&token=` — join/leave/presence, lobby ready, disconnect grace; types in `docs/ws-protocol.md`
- [x] In-memory hub (mutex per room + map mutex), empty rooms closed and dropped so passcodes reuse
- [x] Temporary identity: `X-Player-ID` or `Authorization: Bearer` / `POST /dev/guest` (not phase 4 JWT)
- [x] Phase 3 hooks only: `POST /rooms/:id/start` and WS `submit`/`ready` (while playing) return `NOT_IMPLEMENTED`
- [x] Unit tests: passcode, join rules, hub; WS integration via `httptest`; `make test` (`make test-race` when cgo is available)

## Phase 3 acceptance

- [x] `POST /rooms/:id/start` (host, two players) starts an in-memory match and persists a `matches` row
- [x] WS `submit` / `ready` while playing; moves stay hidden until both Ready or the turn timer fires
- [x] Server-side countdown using the room's `turnSeconds`; `Timeout(roundNo)` so a late timer cannot resolve a later round
- [x] Timeout with no card assigns a random playable move (`auto_picked`); first to `winsToFinish` ends the match
- [x] Persist `rounds` + `round_moves`; update `user_stats` when the match ends
- [x] Mid-match disconnect: grace, pause, then forfeit; `claim_win` if the opponent is already gone; rematch via `play_again` or leave to lobby
- [x] `docs/ws-protocol.md` lists match events; table-driven hub tests plus a two-player WS round; `make test` / `make test-race`
