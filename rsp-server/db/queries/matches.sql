-- name: InsertMatch :one
INSERT INTO matches (room_id)
VALUES ($1)
RETURNING *;

-- name: FinishMatch :exec
UPDATE matches
SET winner_id = $2, ended_at = now()
WHERE id = $1;

-- name: InsertRound :one
INSERT INTO rounds (match_id, round_no, winner_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: InsertRoundMove :exec
INSERT INTO round_moves (round_id, user_id, move, auto_picked)
VALUES ($1, $2, $3, $4);

-- name: ApplyMatchStats :exec
INSERT INTO user_stats (
    user_id, matches_played, wins, losses,
    rock_count, paper_count, scissors_count
)
VALUES ($1, 1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE SET
    matches_played = user_stats.matches_played + 1,
    wins = user_stats.wins + EXCLUDED.wins,
    losses = user_stats.losses + EXCLUDED.losses,
    rock_count = user_stats.rock_count + EXCLUDED.rock_count,
    paper_count = user_stats.paper_count + EXCLUDED.paper_count,
    scissors_count = user_stats.scissors_count + EXCLUDED.scissors_count;

-- name: GetUserStats :one
SELECT * FROM user_stats WHERE user_id = $1;

-- name: GetMatch :one
SELECT * FROM matches WHERE id = $1;
