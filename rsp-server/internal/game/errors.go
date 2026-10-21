package game

import "errors"

var (
	ErrInvalidMove         = errors.New("invalid move")
	ErrInvalidPlayer       = errors.New("invalid player")
	ErrSamePlayer          = errors.New("both sides are the same player")
	ErrUnknownPlayer       = errors.New("player is not in this match")
	ErrInvalidWinsToFinish = errors.New("wins_to_finish must be >= 1")
	ErrInvalidTurnSeconds  = errors.New("turn_seconds must be >= 1")
	ErrAlreadyReady        = errors.New("player already ready this round")
	ErrNoMove              = errors.New("ready requires a submitted move")
	ErrMatchOver           = errors.New("match is over")
	ErrStaleRound          = errors.New("action is for a previous round")
)
