package hub

import "errors"

var (
	ErrNotHost        = errors.New("only the host can start")
	ErrNeedTwo        = errors.New("need two players to start")
	ErrAlreadyPlaying = errors.New("match already in progress")
	ErrNotPlaying     = errors.New("no match in progress")
	ErrMatchOver      = errors.New("match is over")
	ErrOpponentHere   = errors.New("opponent is still connected")
)
