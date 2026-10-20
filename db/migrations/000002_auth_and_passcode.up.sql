-- Password accounts, refresh/reset tokens, and 6-digit room passcodes.
-- Additive on top of 000001; do not edit the applied v1 files.

ALTER TABLE users
    ADD COLUMN username TEXT,
    ADD COLUMN password_hash TEXT;

-- Unique login name, case-insensitive. NULL for guests and Google-only users.
CREATE UNIQUE INDEX users_username_lower_idx
    ON users (lower(username))
    WHERE username IS NOT NULL;

ALTER TABLE users
    ADD CONSTRAINT users_username_format_chk CHECK (
        username IS NULL OR username ~ '^[A-Za-z0-9_]{3,20}$'
    );

-- Username and hash are a pair: password account (or upgraded guest) vs
-- guest / Google-only (both null).
ALTER TABLE users
    ADD CONSTRAINT users_password_pair_chk CHECK (
        (username IS NULL AND password_hash IS NULL)
        OR (username IS NOT NULL AND password_hash IS NOT NULL AND password_hash <> '')
    );

-- Password users may omit email (Forgot password then unavailable).
-- Guests still have no email, username, or password.
ALTER TABLE users DROP CONSTRAINT users_guest_email_chk;

ALTER TABLE users
    ADD CONSTRAINT users_guest_chk CHECK (
        (NOT is_guest)
        OR (
            is_guest
            AND email IS NULL
            AND username IS NULL
            AND password_hash IS NULL
        )
    );

-- Raw tokens never land in the DB — store a hash. Remember me is a longer
-- expires_at, not another column.
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at  TIMESTAMPTZ
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

CREATE TABLE password_reset_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);

ALTER TABLE rooms DROP CONSTRAINT rooms_passcode_check;

ALTER TABLE rooms
    ADD CONSTRAINT rooms_passcode_digits_chk CHECK (passcode ~ '^[0-9]{6}$');
