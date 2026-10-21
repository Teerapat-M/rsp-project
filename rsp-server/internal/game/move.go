package game

// Move is a player's choice for one round.
//
// Timeout is a first-class move so round resolution is one function: every
// player ends the round with exactly one Move, then we compare. The HTTP/WS
// layer never invents a "winner" on its own.
type Move string

const (
	MoveRock     Move = "rock"
	MovePaper    Move = "paper"
	MoveScissors Move = "scissors"
	MoveTimeout  Move = "timeout"
)

// Playable is what a human may submit. Timeout is assigned by the server
// when the turn clock fires and that player never chose a card.
func (m Move) Playable() bool {
	return m == MoveRock || m == MovePaper || m == MoveScissors
}

func (m Move) valid() bool {
	return m.Playable() || m == MoveTimeout
}

// beats is the classic cycle. Timeout never beats anyone; the caller treats
// "real move vs timeout" separately so a throw always beats a no-show.
func (m Move) beats(other Move) bool {
	switch m {
	case MoveRock:
		return other == MoveScissors
	case MovePaper:
		return other == MoveRock
	case MoveScissors:
		return other == MovePaper
	default:
		return false
	}
}

// Outcome is the result of comparing two already-revealed moves.
// Winner is 0 or 1 (the same indexing as Match.Players) when Draw is false.
type Outcome struct {
	Draw   bool
	Winner int
}

// Resolve compares two moves.
//
//	same move (including timeout vs timeout) → draw
//	timeout vs a real throw                 → the throw wins
//	otherwise                               → rock/paper/scissors cycle
func Resolve(a, b Move) (Outcome, error) {
	if !a.valid() || !b.valid() {
		return Outcome{}, ErrInvalidMove
	}
	if a == b {
		return Outcome{Draw: true}, nil
	}
	if a == MoveTimeout {
		return Outcome{Winner: 1}, nil
	}
	if b == MoveTimeout {
		return Outcome{Winner: 0}, nil
	}
	if a.beats(b) {
		return Outcome{Winner: 0}, nil
	}
	return Outcome{Winner: 1}, nil
}
