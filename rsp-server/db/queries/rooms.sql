-- name: InsertRoom :one
INSERT INTO rooms (host_id, passcode, wins_to_finish, turn_seconds)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetRoom :one
SELECT * FROM rooms WHERE id = $1;

-- name: GetOpenRoomByPasscode :one
SELECT * FROM rooms
WHERE passcode = $1 AND status <> 'closed'
LIMIT 1;

-- name: CloseRoom :exec
UPDATE rooms SET status = 'closed' WHERE id = $1;

-- name: SetRoomHost :exec
UPDATE rooms SET host_id = $2 WHERE id = $1;

-- name: SetRoomStatus :exec
UPDATE rooms SET status = $2 WHERE id = $1;

-- name: InsertRoomPlayer :exec
INSERT INTO room_players (room_id, user_id)
VALUES ($1, $2)
ON CONFLICT (room_id, user_id) DO NOTHING;

-- name: DeleteRoomPlayer :exec
DELETE FROM room_players
WHERE room_id = $1 AND user_id = $2;

-- name: ListRoomPlayers :many
SELECT u.id, u.display_name, u.is_guest, u.avatar, u.color, rp.joined_at
FROM room_players rp
INNER JOIN users u ON u.id = rp.user_id
WHERE rp.room_id = $1
ORDER BY rp.joined_at ASC;
