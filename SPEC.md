# Rock-Paper-Scissors — product spec

Online 1v1 Rock-Paper-Scissors. This is a junior full-stack portfolio project: a small, complete game with auth, rooms, a live match, stats, and a React UI.

Target scale is small (~32 concurrent pairs). That number drives one important architecture choice: **active room and match state lives in the Go process memory**. PostgreSQL stores users, auth links, room metadata, and match/round results after a round or match ends — not every keystroke.

---

## Product features

### Auth

- **Google OAuth** for registered accounts.
- **Guest** play: the player picks a display name, no email. Guests are still real `users` rows (`is_guest = true`, `email` null).
- Both paths mint a **JWT**. The server trusts the JWT, not the client, for who is in a room.

Guests do **not** get an `auth_providers` row. That table is for external IdPs (Google in this project). A guest who later signs in with Google is a later-phase concern; phase 4 ships the two paths as separate users.

### Rooms

- Host creates a room with:
  - `wins_to_finish` — first player to this many **round wins** ends the match (draws do not count).
  - `turn_seconds` — countdown length for each round.
- The server generates a **passcode**. Passcodes are unique among rooms that are not `closed` (partial unique index). After close, the code may be reused.
- Others join by passcode. A room is two players.

### Gameplay

1. Each player drags one of rock / paper / scissors into a drop slot, then presses **Ready**.
2. The **server is authoritative**. It stores both choices but **does not reveal** a player's move to the opponent until the round resolves.
3. The round resolves when **both are Ready**, or when the **turn timer ends**.
4. No submitted move when the timer ends → `timeout`. Timeout **loses the round** against a real move. **Both timeout → draw**.
5. Score is round wins. Play continues until one player reaches `wins_to_finish`.

A player may change their card **until** they press Ready. Ready without a card is rejected. After Ready, the move is locked.

The clock is owned by the server process (WebSocket layer), not by the pure rules package. The rules package receives `Timeout(roundNo)` so a late timer from round N cannot resolve round N+1.

### Stats and history

Per user, persisted at round/match end:

- matches played / wins / losses
- how often they played rock / paper / scissors (timeouts do not increment a throw count)
- match history (rooms → matches → rounds → moves)

---

## Data model

PostgreSQL. UUIDs for public ids. See `db/migrations/000001_init.up.sql`.

| Table | Role |
| --- | --- |
| `users` | Account. `email` null iff guest. |
| `auth_providers` | Google (and future IdPs): `(provider, provider_user_id)` unique. |
| `user_stats` | 1:1 with user, updated when a match ends. |
| `rooms` | Host, passcode, settings, `waiting` / `playing` / `closed`. |
| `room_players` | Who joined which room. |
| `matches` | One game series in a room. `winner_id` null until someone wins (or stays null if abandoned). |
| `rounds` | `winner_id` null means draw. |
| `round_moves` | `rock` / `paper` / `scissors` / `timeout`. |

Adjustments from the original draft (kept small):

- `matches.winner_id` is nullable so an in-progress or abandoned match can be stored.
- Check constraints: `wins_to_finish >= 1`, `turn_seconds >= 1`, guest XOR email.
- Indexes on FKs used by history queries (`matches.room_id`, `rounds.match_id`, `auth_providers.user_id`).
- No extra columns (`updated_at`, chat, rematch). Add them if a later phase needs them.

Active gameplay (hidden moves, ready flags, remaining seconds) is **not** in these tables. It lives on the in-memory match in the Go server.

---

## Architecture (whole project)

```
Browser (React + Vite + dnd-kit, /web)
    │  REST: auth, create/join room, stats
    │  WebSocket: match events
    ▼
Go API (gin) + WS hub
    │  in-memory rooms / matches   ← internal/game is the rules engine
    ▼
PostgreSQL   (users, rooms, persisted results)
```

Stack: Go 1.22+, gin, gorilla/websocket, PostgreSQL, sqlc, golang-migrate, Docker Compose.

**gorilla/websocket** rather than coder/websocket: more examples a junior can cite, same job at this scale.

**sqlc** rather than an ORM: SQL stays visible (migrations + `db/queries`), generated Go is boring and typed.

**In-memory match state** is fine at ~32 pairs. The ceiling is "one process holds all live games." Phase 6 can stay a single VM; Redis/multi-instance is an upgrade path, not a requirement.

Layout:

```
cmd/server          process entry (HTTP)
internal/game       pure rules — no I/O, no clocks
internal/sqlcdb     sqlc output (later phases)
db/migrations       golang-migrate
db/queries          sqlc input (later phases)
web/                React app (phase 5)
```

---

## Six-phase roadmap

| Phase | Ships | Does not ship |
| --- | --- | --- |
| **1 — logic + scaffold** (this PR) | `SPEC.md`, idiomatic repo, Compose + Postgres + initial schema, `internal/game` + table-driven tests | HTTP game API, WS, auth, UI |
| **2 — rooms / passcode / websocket** | Create/join room by passcode, WS connect with JWT placeholder or session id, presence in a room | Real Google/guest login, countdown UI |
| **3 — countdown / ready / resolve** | Server timer, submit + ready, hide moves, resolve, `wins_to_finish`, persist round/match rows | Auth, polished UI |
| **4 — guest then Google auth** | Guest display name + JWT, Google OAuth + JWT, `users` / `auth_providers` | Stats page, drag-drop |
| **5 — stats / history + React drag-drop** | `user_stats` + history API, React + Vite in `/web`, dnd-kit cards → drop slot → Ready | Production hosting |
| **6 — deploy** | Dockerfile, Compose or a small VM, env-based config, README for a stranger to run it | Extra game modes |

Each phase is one PR. Do not fold later phases into this one.

---

## Phase 1 acceptance

- [x] Spec and roadmap in `SPEC.md`
- [x] `cmd/`, `internal/`, `go.mod`, `Makefile`, Compose Postgres, migrations, `.env.example`, README
- [x] `internal/game` covers move resolution, timeout, round/match state, `wins_to_finish`
- [x] Table-driven unit tests, `go test ./...` passing
- [x] `docker compose up` starts Postgres and applies migrations
