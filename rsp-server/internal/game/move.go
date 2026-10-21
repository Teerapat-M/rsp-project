package game

// Move is a player's choice for one round.
type Move string

const (
	MoveRock     Move = "rock"
	MovePaper    Move = "paper"
	MoveScissors Move = "scissors"
)

// Playable is what a human may submit, and what the server assigns on timeout.
func (m Move) Playable() bool {
	return m == MoveRock || m == MovePaper || m == MoveScissors
}

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

// Resolve compares two playable moves.
//
//	same move → draw
//	otherwise → rock/paper/scissors cycle
func Resolve(a, b Move) (Outcome, error) {
	if !a.Playable() || !b.Playable() {
		return Outcome{}, ErrInvalidMove
	}
	if a == b {
		return Outcome{Draw: true}, nil
	}
	if a.beats(b) {
		return Outcome{Winner: 0}, nil
	}
	return Outcome{Winner: 1}, nil
}
