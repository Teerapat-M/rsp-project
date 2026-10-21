-- Initial schema for users, rooms, and persisted match results.
-- Live gameplay (hidden moves, ready flags, countdown) stays in Go memory;
-- these tables are written at round/match boundaries.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name  TEXT NOT NULL CHECK (display_name <> ''),
    email         TEXT UNIQUE,
    is_guest      BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_guest_email_chk CHECK (
        (is_guest AND email IS NULL) OR (NOT is_guest AND email IS NOT NULL)
    )
);

-- External identity providers (Google in phase 4). Guests have no row here.
CREATE TABLE auth_providers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider          TEXT NOT NULL,
    provider_user_id  TEXT NOT NULL,
    UNIQUE (provider, provider_user_id)
);

CREATE INDEX auth_providers_user_id_idx ON auth_providers (user_id);

CREATE TABLE user_stats (
    user_id         UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    matches_played  INT NOT NULL DEFAULT 0 CHECK (matches_played >= 0),
    wins            INT NOT NULL DEFAULT 0 CHECK (wins >= 0),
    losses          INT NOT NULL DEFAULT 0 CHECK (losses >= 0),
    rock_count      INT NOT NULL DEFAULT 0 CHECK (rock_count >= 0),
    paper_count     INT NOT NULL DEFAULT 0 CHECK (paper_count >= 0),
    scissors_count  INT NOT NULL DEFAULT 0 CHECK (scissors_count >= 0)
);

CREATE TYPE room_status AS ENUM ('waiting', 'playing', 'closed');

CREATE TABLE rooms (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id         UUID NOT NULL REFERENCES users (id),
    passcode        TEXT NOT NULL CHECK (passcode <> ''),
    wins_to_finish  INT NOT NULL CHECK (wins_to_finish >= 1),
    turn_seconds    INT NOT NULL CHECK (turn_seconds >= 1),
    status          room_status NOT NULL DEFAULT 'waiting',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A passcode can be reused after the room closes; two live rooms cannot share one.
CREATE UNIQUE INDEX rooms_passcode_active_idx
    ON rooms (passcode)
    WHERE status <> 'closed';

CREATE INDEX rooms_host_id_idx ON rooms (host_id);

CREATE TABLE room_players (
    room_id    UUID NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id),
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id)
);

CREATE TABLE matches (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id     UUID NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    winner_id   UUID REFERENCES users (id),
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at    TIMESTAMPTZ
);

CREATE INDEX matches_room_id_idx ON matches (room_id);

CREATE TABLE rounds (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id   UUID NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    round_no   INT NOT NULL CHECK (round_no >= 1),
    winner_id  UUID REFERENCES users (id),
    UNIQUE (match_id, round_no)
);

CREATE INDEX rounds_match_id_idx ON rounds (match_id);

CREATE TYPE move_kind AS ENUM ('rock', 'paper', 'scissors', 'timeout');

CREATE TABLE round_moves (
    round_id  UUID NOT NULL REFERENCES rounds (id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users (id),
    move      move_kind NOT NULL,
    PRIMARY KEY (round_id, user_id)
);
