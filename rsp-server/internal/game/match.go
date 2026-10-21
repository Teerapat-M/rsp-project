// Package game is the in-memory rules engine for a two-player match.
//
// It has no I/O and no time.Ticker. The WebSocket layer (phase 3) owns the
// wall clock and calls Timeout(roundNo) when a turn expires. That keeps every
// rule testable as "given these inputs, this is the next state."
//
// This package is not safe for concurrent use. The room hub should serialize
// calls for a given match (one goroutine per room, or a mutex around it).
package game

// PlayerID is whoever the server considers a player (a user UUID later).
type PlayerID string

// Status is the match-level state: still playing rounds, or someone hit wins_to_finish.
type Status int

const (
	StatusPlaying Status = iota
	StatusFinished
)

func (s Status) String() string {
	switch s {
	case StatusPlaying:
		return "playing"
	case StatusFinished:
		return "finished"
	default:
		return "unknown"
	}
}

// Match is first-to-N round wins. Draws do not add to either score.
type Match struct {
	winsNeeded  int
	turnSeconds int
	players     [2]PlayerID
	scores      [2]int
	roundNo     int
	status      Status
	round       *openRound
	history     []ResolvedRound
}

// openRound is the in-progress round. Moves stay pointer-nil until submitted
// so "no card yet" is distinct from timeout (which is only applied at resolve).
type openRound struct {
	moves [2]*Move
	ready [2]bool
}

// ResolvedRound is a finished round with both moves revealed. Persist this
// (phase 3+) — it is the record the opponent was not allowed to see mid-round.
type ResolvedRound struct {
	Number int
	Moves  map[PlayerID]Move
	Winner PlayerID // empty if Draw
	Draw   bool
}

// NewMatch starts round 1. winsToFinish is "first to N wins", not "play N games".
func NewMatch(a, b PlayerID, winsToFinish, turnSeconds int) (*Match, error) {
	if a == "" || b == "" {
		return nil, ErrInvalidPlayer
	}
	if a == b {
		return nil, ErrSamePlayer
	}
	if winsToFinish < 1 {
		return nil, ErrInvalidWinsToFinish
	}
	if turnSeconds < 1 {
		return nil, ErrInvalidTurnSeconds
	}
	m := &Match{
		winsNeeded:  winsToFinish,
		turnSeconds: turnSeconds,
		players:     [2]PlayerID{a, b},
	}
	m.openRound()
	return m, nil
}

func (m *Match) openRound() {
	m.roundNo++
	m.round = &openRound{}
}

func (m *Match) side(id PlayerID) (int, error) {
	for i, p := range m.players {
		if p == id {
			return i, nil
		}
	}
	return 0, ErrUnknownPlayer
}

func (m *Match) guardRound(roundNo int) error {
	if m.status == StatusFinished {
		return ErrMatchOver
	}
	if roundNo != m.roundNo {
		return ErrStaleRound
	}
	return nil
}

// Submit stores a hidden playable move. Allowed until this player presses Ready.
// Passing the current round number rejects delayed packets from the previous round.
func (m *Match) Submit(id PlayerID, roundNo int, move Move) error {
	if err := m.guardRound(roundNo); err != nil {
		return err
	}
	if !move.Playable() {
		return ErrInvalidMove
	}
	i, err := m.side(id)
	if err != nil {
		return err
	}
	if m.round.ready[i] {
		return ErrAlreadyReady
	}
	copied := move
	m.round.moves[i] = &copied
	return nil
}

// Ready locks this player's move. When both are ready the round resolves and
// the returned pointer is non-nil. Otherwise it is nil (still waiting).
func (m *Match) Ready(id PlayerID, roundNo int) (*ResolvedRound, error) {
	if err := m.guardRound(roundNo); err != nil {
		return nil, err
	}
	i, err := m.side(id)
	if err != nil {
		return nil, err
	}
	if m.round.ready[i] {
		return nil, ErrAlreadyReady
	}
	if m.round.moves[i] == nil {
		return nil, ErrNoMove
	}
	m.round.ready[i] = true
	if m.round.ready[0] && m.round.ready[1] {
		return m.resolve(), nil
	}
	return nil, nil
}

// Timeout ends the current round because the turn clock fired.
// Anyone without a move is assigned timeout. Submitted-but-not-ready moves still count
// ("no move = timeout", not "not ready = timeout").
func (m *Match) Timeout(roundNo int) (*ResolvedRound, error) {
	if err := m.guardRound(roundNo); err != nil {
		return nil, err
	}
	return m.resolve(), nil
}

func (m *Match) resolve() *ResolvedRound {
	timeout := MoveTimeout
	var pair [2]Move
	for i := 0; i < 2; i++ {
		if m.round.moves[i] == nil {
			m.round.moves[i] = &timeout
		}
		pair[i] = *m.round.moves[i]
	}
	out, err := Resolve(pair[0], pair[1])
	if err != nil {
		// Only reachable if this package handed Resolve an invalid move —
		// that would be a bug in Submit/Timeout, not in caller input.
		panic(err)
	}

	rr := ResolvedRound{
		Number: m.roundNo,
		Moves: map[PlayerID]Move{
			m.players[0]: pair[0],
			m.players[1]: pair[1],
		},
		Draw: out.Draw,
	}
	if !out.Draw {
		rr.Winner = m.players[out.Winner]
		m.scores[out.Winner]++
	}
	m.history = append(m.history, rr)

	if m.scores[0] >= m.winsNeeded || m.scores[1] >= m.winsNeeded {
		m.status = StatusFinished
		m.round = nil
	} else {
		m.openRound()
	}
	return &rr
}

func (m *Match) Status() Status           { return m.status }
func (m *Match) RoundNumber() int         { return m.roundNo }
func (m *Match) WinsNeeded() int          { return m.winsNeeded }
func (m *Match) TurnSeconds() int         { return m.turnSeconds }
func (m *Match) Players() [2]PlayerID     { return m.players }
func (m *Match) History() []ResolvedRound { return append([]ResolvedRound(nil), m.history...) }

func (m *Match) Score(id PlayerID) (int, error) {
	i, err := m.side(id)
	if err != nil {
		return 0, err
	}
	return m.scores[i], nil
}

// Forfeit ends the match because loser disconnected past grace (or the opponent
// claimed the win). Completed rounds stay as-is; the remaining player is the
// winner. Their score is raised to winsNeeded so the result screen is not 0–0.
func (m *Match) Forfeit(loser PlayerID) error {
	if m.status == StatusFinished {
		return ErrMatchOver
	}
	i, err := m.side(loser)
	if err != nil {
		return err
	}
	opp := 1 - i
	if m.scores[opp] < m.winsNeeded {
		m.scores[opp] = m.winsNeeded
	}
	m.status = StatusFinished
	m.round = nil
	return nil
}

// Winner reports the player who reached wins_to_finish. ok is false while playing
// (and would stay false if we ever persisted an abandoned match with no winner).
func (m *Match) Winner() (PlayerID, bool) {
	if m.status != StatusFinished {
		return "", false
	}
	if m.scores[0] > m.scores[1] {
		return m.players[0], true
	}
	return m.players[1], true
}

// PlayerView is everything one player may see. The opponent's current card is
// omitted on purpose — that is the whole point of server-authoritative play.
type PlayerView struct {
	Status        Status
	RoundNumber   int
	WinsNeeded    int
	TurnSeconds   int
	Scores        map[PlayerID]int
	YourMove      Move // empty if they have not submitted this round
	YourReady     bool
	Opponent      PlayerID
	OpponentReady bool
	Winner        PlayerID
	Finished      bool
	History       []ResolvedRound
}

func (m *Match) ViewFor(id PlayerID) (PlayerView, error) {
	i, err := m.side(id)
	if err != nil {
		return PlayerView{}, err
	}
	opp := 1 - i
	v := PlayerView{
		Status:      m.status,
		RoundNumber: m.roundNo,
		WinsNeeded:  m.winsNeeded,
		TurnSeconds: m.turnSeconds,
		Scores:      map[PlayerID]int{m.players[0]: m.scores[0], m.players[1]: m.scores[1]},
		Opponent:    m.players[opp],
		Finished:    m.status == StatusFinished,
		History:     m.History(),
	}
	if w, ok := m.Winner(); ok {
		v.Winner = w
	}
	if m.round != nil {
		if m.round.moves[i] != nil {
			v.YourMove = *m.round.moves[i]
		}
		v.YourReady = m.round.ready[i]
		v.OpponentReady = m.round.ready[opp]
	}
	return v, nil
}
