-- name: InsertGuest :one
-- Temporary guest upsert used until phase 4 JWT auth.
INSERT INTO users (id, display_name, is_guest)
VALUES ($1, $2, true)
ON CONFLICT (id) DO UPDATE
SET display_name = CASE
    WHEN users.is_guest
         AND EXCLUDED.display_name <> ''
         AND EXCLUDED.display_name <> 'Guest'
    THEN EXCLUDED.display_name
    ELSE users.display_name
END
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;
