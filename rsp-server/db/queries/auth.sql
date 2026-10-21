-- name: InsertUser :one
INSERT INTO users (display_name, email, is_guest, username, password_hash, avatar, color)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE lower(username) = lower($1)
LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE lower(email) = lower($1)
LIMIT 1;

-- name: UpgradeGuest :one
UPDATE users
SET username = $2,
    password_hash = $3,
    email = $4,
    is_guest = false,
    display_name = COALESCE(NULLIF($5, ''), display_name)
WHERE id = $1 AND is_guest = true
RETURNING *;

-- name: SetPasswordHash :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: InsertAuthProvider :one
INSERT INTO auth_providers (user_id, provider, provider_user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAuthProvider :one
SELECT * FROM auth_providers
WHERE provider = $1 AND provider_user_id = $2;

-- name: InsertRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE token_hash = $1 AND revoked_at IS NULL;

-- name: RevokeUserRefreshTokens :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: InsertPasswordReset :one
INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPasswordReset :one
SELECT * FROM password_reset_tokens WHERE token_hash = $1;

-- name: MarkPasswordResetUsed :exec
UPDATE password_reset_tokens SET used_at = now()
WHERE id = $1 AND used_at IS NULL;
