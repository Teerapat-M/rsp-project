-- name: CountEndedMatchesForUser :one
SELECT count(*)
FROM matches m
INNER JOIN room_players rp ON rp.room_id = m.room_id AND rp.user_id = $1
WHERE m.ended_at IS NOT NULL;

-- name: ListEndedMatchesForUser :many
SELECT m.id, m.room_id, m.winner_id, m.started_at, m.ended_at,
       r.wins_to_finish, r.turn_seconds
FROM matches m
INNER JOIN rooms r ON r.id = m.room_id
INNER JOIN room_players rp ON rp.room_id = m.room_id AND rp.user_id = $1
WHERE m.ended_at IS NOT NULL
ORDER BY m.ended_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMatchForUser :one
SELECT m.id, m.room_id, m.winner_id, m.started_at, m.ended_at,
       r.wins_to_finish, r.turn_seconds
FROM matches m
INNER JOIN rooms r ON r.id = m.room_id
INNER JOIN room_players rp ON rp.room_id = m.room_id AND rp.user_id = $1
WHERE m.id = $2 AND m.ended_at IS NOT NULL;

-- name: ListMatchMoves :many
SELECT rd.round_no, rd.winner_id, rm.user_id, rm.move
FROM rounds rd
INNER JOIN round_moves rm ON rm.round_id = rd.id
WHERE rd.match_id = $1
ORDER BY rd.round_no ASC, rm.user_id ASC;

-- name: GetMatchOpponent :one
SELECT u.id, u.display_name, u.avatar, u.color
FROM room_players rp
INNER JOIN users u ON u.id = rp.user_id
WHERE rp.room_id = $1 AND rp.user_id <> $2
ORDER BY rp.joined_at ASC
LIMIT 1;

-- name: CountPickWins :many
SELECT rm.move, count(*)::int AS n
FROM rounds rd
INNER JOIN round_moves rm ON rm.round_id = rd.id
WHERE rm.user_id = $1 AND rd.winner_id = $1 AND rm.move <> 'timeout'
GROUP BY rm.move;

-- name: ListWinnerIDsForUser :many
SELECT m.winner_id, m.ended_at
FROM matches m
INNER JOIN room_players rp ON rp.room_id = m.room_id AND rp.user_id = $1
WHERE m.ended_at IS NOT NULL
ORDER BY m.ended_at ASC;
