package game

import (
	"errors"
	"testing"
)

const (
	alice PlayerID = "alice"
	bob   PlayerID = "bob"
)

func newTestMatch(t *testing.T, wins, seconds int) *Match {
	t.Helper()
	m, err := NewMatch(alice, bob, wins, seconds)
	if err != nil {
		t.Fatalf("NewMatch: %v", err)
	}
	return m
}

func mustSubmit(t *testing.T, m *Match, id PlayerID, round int, move Move) {
	t.Helper()
	if err := m.Submit(id, round, move); err != nil {
		t.Fatalf("Submit(%s, %s): %v", id, move, err)
	}
}

func TestNewMatchValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		a, b    PlayerID
		wins    int
		seconds int
		want    error
	}{
		{name: "ok", a: alice, b: bob, wins: 3, seconds: 10},
		{name: "empty a", a: "", b: bob, wins: 1, seconds: 1, want: ErrInvalidPlayer},
		{name: "empty b", a: alice, b: "", wins: 1, seconds: 1, want: ErrInvalidPlayer},
		{name: "same player", a: alice, b: alice, wins: 1, seconds: 1, want: ErrSamePlayer},
		{name: "wins 0", a: alice, b: bob, wins: 0, seconds: 10, want: ErrInvalidWinsToFinish},
		{name: "wins negative", a: alice, b: bob, wins: -1, seconds: 10, want: ErrInvalidWinsToFinish},
		{name: "seconds 0", a: alice, b: bob, wins: 1, seconds: 0, want: ErrInvalidTurnSeconds},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewMatch(tt.a, tt.b, tt.wins, tt.seconds)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBothReadyResolvesHiddenMoves(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 2, 15)

	mustSubmit(t, m, alice, 1, MoveRock)
	mustSubmit(t, m, bob, 1, MoveScissors)

	// Opponent's card is hidden until resolve.
	view, err := m.ViewFor(alice)
	if err != nil {
		t.Fatal(err)
	}
	if view.YourMove != MoveRock {
		t.Fatalf("alice should see her own move, got %q", view.YourMove)
	}
	if view.OpponentReady {
		t.Fatal("bob is not ready yet")
	}

	got, err := m.Ready(alice, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("one ready should not resolve")
	}

	got, err = m.Ready(bob, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("both ready should resolve")
	}
	if got.Draw || got.Winner != alice || got.Moves[bob] != MoveScissors {
		t.Fatalf("resolved round = %+v", got)
	}

	if s, _ := m.Score(alice); s != 1 {
		t.Fatalf("alice score = %d", s)
	}
	if m.RoundNumber() != 2 {
		t.Fatalf("next round should be 2, got %d", m.RoundNumber())
	}
}

func TestSubmitCanChangeBeforeReady(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 1, 10)
	mustSubmit(t, m, alice, 1, MoveRock)
	mustSubmit(t, m, alice, 1, MovePaper) // changed mind
	mustSubmit(t, m, bob, 1, MoveRock)
	rr, err := m.Ready(alice, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rr != nil {
		t.Fatal("bob not ready")
	}
	rr, err = m.Ready(bob, 1)
	if err != nil || rr == nil {
		t.Fatalf("rr=%v err=%v", rr, err)
	}
	if rr.Moves[alice] != MovePaper {
		t.Fatalf("wanted last submit (paper), got %s", rr.Moves[alice])
	}
}

func TestCannotSubmitAfterReady(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 3, 10)
	mustSubmit(t, m, alice, 1, MoveRock)
	if _, err := m.Ready(alice, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.Submit(alice, 1, MovePaper); !errors.Is(err, ErrAlreadyReady) {
		t.Fatalf("err = %v, want ErrAlreadyReady", err)
	}
}

func TestReadyRequiresMove(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 1, 10)
	if _, err := m.Ready(alice, 1); !errors.Is(err, ErrNoMove) {
		t.Fatalf("err = %v, want ErrNoMove", err)
	}
}

func TestReadyTwice(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 3, 10)
	mustSubmit(t, m, alice, 1, MoveRock)
	if _, err := m.Ready(alice, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Ready(alice, 1); !errors.Is(err, ErrAlreadyReady) {
		t.Fatalf("err = %v, want ErrAlreadyReady", err)
	}
}

func TestTimeoutUsesSubmittedMoveNotReady(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 2, 10)
	mustSubmit(t, m, alice, 1, MoveScissors)
	// bob never acts
	rr, err := m.Timeout(1)
	if err != nil {
		t.Fatal(err)
	}
	if rr.Moves[alice] != MoveScissors || rr.Moves[bob] != MoveTimeout {
		t.Fatalf("moves = %+v", rr.Moves)
	}
	if rr.Winner != alice {
		t.Fatalf("winner = %s, want alice (bob timed out)", rr.Winner)
	}
}

func TestTimeoutBothEmptyIsDraw(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 2, 10)
	rr, err := m.Timeout(1)
	if err != nil {
		t.Fatal(err)
	}
	if !rr.Draw || rr.Moves[alice] != MoveTimeout || rr.Moves[bob] != MoveTimeout {
		t.Fatalf("want double timeout draw, got %+v", rr)
	}
	if m.Status() != StatusPlaying {
		t.Fatal("draw must not finish a first-to-2 match")
	}
	if m.RoundNumber() != 2 {
		t.Fatalf("round = %d, want 2", m.RoundNumber())
	}
}

func TestTimeoutOneReadyOneEmpty(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 2, 10)
	mustSubmit(t, m, bob, 1, MovePaper)
	if _, err := m.Ready(bob, 1); err != nil {
		t.Fatal(err)
	}
	rr, err := m.Timeout(1)
	if err != nil {
		t.Fatal(err)
	}
	if rr.Winner != bob || rr.Moves[alice] != MoveTimeout {
		t.Fatalf("got %+v", rr)
	}
}

func TestStaleRoundRejected(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 3, 10)
	mustSubmit(t, m, alice, 1, MoveRock)
	mustSubmit(t, m, bob, 1, MovePaper)
	if _, err := m.Ready(alice, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Ready(bob, 1); err != nil {
		t.Fatal(err)
	}
	// Round 1 is done (bob won). A late timer or packet must not touch round 2.
	if err := m.Submit(alice, 1, MoveRock); !errors.Is(err, ErrStaleRound) {
		t.Fatalf("submit stale: %v", err)
	}
	if _, err := m.Ready(alice, 1); !errors.Is(err, ErrStaleRound) {
		t.Fatalf("ready stale: %v", err)
	}
	if _, err := m.Timeout(1); !errors.Is(err, ErrStaleRound) {
		t.Fatalf("timeout stale: %v", err)
	}
	if m.RoundNumber() != 2 {
		t.Fatalf("round = %d", m.RoundNumber())
	}
}

func TestFirstToNWinsAndRejectsFurtherPlay(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 2, 8)

	play := func(round int, a, b Move) {
		t.Helper()
		mustSubmit(t, m, alice, round, a)
		mustSubmit(t, m, bob, round, b)
		if _, err := m.Ready(alice, round); err != nil {
			t.Fatal(err)
		}
		rr, err := m.Ready(bob, round)
		if err != nil || rr == nil {
			t.Fatalf("resolve round %d: rr=%v err=%v", round, rr, err)
		}
	}

	play(1, MoveRock, MoveScissors)  // alice 1-0
	play(2, MovePaper, MovePaper)    // draw, still 1-0, now round 3
	play(3, MoveScissors, MovePaper) // alice 2-0, match over

	if m.Status() != StatusFinished {
		t.Fatalf("status = %s", m.Status())
	}
	w, ok := m.Winner()
	if !ok || w != alice {
		t.Fatalf("winner = %s ok=%v", w, ok)
	}
	if sa, _ := m.Score(alice); sa != 2 {
		t.Fatalf("alice = %d", sa)
	}
	if sb, _ := m.Score(bob); sb != 0 {
		t.Fatalf("bob = %d", sb)
	}
	if len(m.History()) != 3 {
		t.Fatalf("history len = %d", len(m.History()))
	}

	if err := m.Submit(alice, m.RoundNumber(), MoveRock); !errors.Is(err, ErrMatchOver) {
		t.Fatalf("submit after finish: %v", err)
	}
	if _, err := m.Timeout(m.RoundNumber()); !errors.Is(err, ErrMatchOver) {
		t.Fatalf("timeout after finish: %v", err)
	}
}

func TestBobCanWinMatch(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 1, 5)
	mustSubmit(t, m, alice, 1, MoveRock)
	mustSubmit(t, m, bob, 1, MovePaper)
	_, _ = m.Ready(alice, 1)
	rr, err := m.Ready(bob, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rr.Winner != bob {
		t.Fatalf("winner = %s", rr.Winner)
	}
	w, ok := m.Winner()
	if !ok || w != bob {
		t.Fatalf("match winner = %s ok=%v", w, ok)
	}
}

func TestUnknownPlayerAndInvalidMove(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 1, 10)
	if err := m.Submit("carol", 1, MoveRock); !errors.Is(err, ErrUnknownPlayer) {
		t.Fatalf("unknown: %v", err)
	}
	if err := m.Submit(alice, 1, MoveTimeout); !errors.Is(err, ErrInvalidMove) {
		t.Fatalf("timeout submit: %v", err)
	}
	if err := m.Submit(alice, 1, Move("nuke")); !errors.Is(err, ErrInvalidMove) {
		t.Fatalf("junk submit: %v", err)
	}
	if _, err := m.ViewFor("carol"); !errors.Is(err, ErrUnknownPlayer) {
		t.Fatalf("view: %v", err)
	}
	if _, err := m.Score("carol"); !errors.Is(err, ErrUnknownPlayer) {
		t.Fatalf("score: %v", err)
	}
}

func TestViewForHidesOpponentMove(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 3, 12)
	mustSubmit(t, m, alice, 1, MoveRock)
	mustSubmit(t, m, bob, 1, MovePaper)

	aliceView, _ := m.ViewFor(alice)
	if aliceView.YourMove != MoveRock {
		t.Fatal("alice must see her card")
	}
	bobView, _ := m.ViewFor(bob)
	if bobView.YourMove != MovePaper {
		t.Fatal("bob must see his card")
	}
	// PlayerView has no OpponentMove field on purpose; history is empty until resolve.
	if len(aliceView.History) != 0 {
		t.Fatal("nothing revealed yet")
	}

	_, _ = m.Ready(alice, 1)
	_, _ = m.Ready(bob, 1)

	aliceView, _ = m.ViewFor(alice)
	if len(aliceView.History) != 1 || aliceView.History[0].Moves[bob] != MovePaper {
		t.Fatalf("after resolve, history should reveal bob's paper: %+v", aliceView.History)
	}
	if aliceView.YourMove != "" {
		t.Fatal("after resolve we opened round 2; alice has no new card")
	}
}

func TestTurnSecondsAndPlayersExposed(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 5, 30)
	if m.TurnSeconds() != 30 || m.WinsNeeded() != 5 {
		t.Fatalf("config not stored")
	}
	ps := m.Players()
	if ps[0] != alice || ps[1] != bob {
		t.Fatalf("players = %v", ps)
	}
}

func TestDrawDoesNotAwardWin(t *testing.T) {
	t.Parallel()
	m := newTestMatch(t, 1, 10)
	mustSubmit(t, m, alice, 1, MoveRock)
	mustSubmit(t, m, bob, 1, MoveRock)
	_, _ = m.Ready(alice, 1)
	rr, err := m.Ready(bob, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !rr.Draw || rr.Winner != "" {
		t.Fatalf("%+v", rr)
	}
	if m.Status() != StatusPlaying {
		t.Fatal("first-to-1 should continue after a draw")
	}
	sa, _ := m.Score(alice)
	sb, _ := m.Score(bob)
	if sa != 0 || sb != 0 {
		t.Fatalf("scores %d-%d", sa, sb)
	}
}
