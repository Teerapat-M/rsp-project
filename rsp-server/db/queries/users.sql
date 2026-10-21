-- name: InsertGuest :one
INSERT INTO users (id, display_name, is_guest, avatar, color)
VALUES ($1, $2, true, $3, $4)
ON CONFLICT (id) DO UPDATE
SET display_name = CASE
    WHEN users.is_guest
         AND EXCLUDED.display_name <> ''
         AND EXCLUDED.display_name <> 'Guest'
    THEN EXCLUDED.display_name
    ELSE users.display_name
END,
    avatar = CASE WHEN users.is_guest AND EXCLUDED.avatar <> '' THEN EXCLUDED.avatar ELSE users.avatar END,
    color = CASE WHEN users.is_guest AND EXCLUDED.color <> '' THEN EXCLUDED.color ELSE users.color END
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;
