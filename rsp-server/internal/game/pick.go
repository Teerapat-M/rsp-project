package game

import crand "crypto/rand"

// Pick returns a playable move. Tests inject a deterministic Pick so a timeout
// is not a lottery. Production uses CryptoPick.
type Pick func() Move

var playableMoves = [3]Move{MoveRock, MovePaper, MoveScissors}

// CryptoPick chooses rock, paper, or scissors with crypto/rand.
func CryptoPick() Move {
	var b [1]byte
	if _, err := crand.Read(b[:]); err != nil {
		return MoveRock
	}
	return playableMoves[int(b[0])%len(playableMoves)]
}

// SetPick replaces the timeout picker. Ignored if p is nil.
func (m *Match) SetPick(p Pick) {
	if p != nil {
		m.pick = p
	}
}
