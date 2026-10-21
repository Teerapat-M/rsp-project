package hub

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/game"
	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
)

type fakeClock struct {
	mu   sync.Mutex
	jobs []*scheduled
}

type scheduled struct {
	d       time.Duration
	f       func()
	stopped atomic.Bool
}

func (c *fakeClock) After(d time.Duration, f func()) func() bool {
	j := &scheduled{d: d, f: f}
	c.mu.Lock()
	c.jobs = append(c.jobs, j)
	c.mu.Unlock()
	return func() bool {
		return !j.stopped.Swap(true)
	}
}

func (c *fakeClock) Fire(d time.Duration) int {
	c.mu.Lock()
	var run []func()
	for _, j := range c.jobs {
		if j.d == d && !j.stopped.Swap(true) {
			run = append(run, j.f)
		}
	}
	c.mu.Unlock()
	for _, f := range run {
		f()
	}
	return len(run)
}

func playHub(t *testing.T, clk *fakeClock) (*Hub, *rooms.Memory) {
	t.Helper()
	st := rooms.NewMemory()
	h := New(st, Config{
		Grace:       50 * time.Millisecond,
		After:       clk.After,
		ResultPause: 3 * time.Second,
	})
	return h, st
}

func twoSeated(t *testing.T, h *Hub) (rooms.Room, ident.Player, ident.Player, *client, *client) {
	t.Helper()
	ctx := context.Background()
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
	room, err := h.Create(ctx, host, 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Join(ctx, guest, room.Passcode); err != nil {
		t.Fatal(err)
	}
	s := h.get(room.ID)
	a := seat(h, s, host)
	b := seat(h, s, guest)
	return room, host, guest, a, b
}

func seat(h *Hub, s *session, p ident.Player) *client {
	c := &client{hub: h, sess: s, player: p, send: make(chan []byte, 64)}
	s.mu.Lock()
	m, ok := s.findLocked(p.ID)
	if !ok {
		s.mu.Unlock()
		panic("player not in room")
	}
	m.conns[c] = struct{}{}
	s.mu.Unlock()
	return c
}

func drain(c *client) []map[string]any {
	var out []map[string]any
	for {
		select {
		case b, ok := <-c.send:
			if !ok {
				return out
			}
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			out = append(out, m)
		default:
			return out
		}
	}
}

func lastType(msgs []map[string]any, typ string) map[string]any {
	var found map[string]any
	for _, m := range msgs {
		if m["type"] == typ {
			found = m
		}
	}
	return found
}

func viewOf(msg map[string]any) map[string]any {
	if msg == nil {
		return nil
	}
	v, _ := msg["view"].(map[string]any)
	return v
}

func TestStartRequiresHostAndTwoPlayers(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, _ := playHub(t, clk)
	ctx := context.Background()
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
	room, err := h.Create(ctx, host, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.StartMatch(room.ID, host.ID); !errors.Is(err, ErrNeedTwo) {
		t.Fatalf("need two: %v", err)
	}
	if _, err := h.Join(ctx, guest, room.Passcode); err != nil {
		t.Fatal(err)
	}
	if err := h.StartMatch(room.ID, guest.ID); !errors.Is(err, ErrNotHost) {
		t.Fatalf("guest start: %v", err)
	}
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.StartMatch(room.ID, host.ID); !errors.Is(err, ErrAlreadyPlaying) {
		t.Fatalf("double start: %v", err)
	}
}

func TestSubmitReadyHidesMovesThenResolves(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, st := playHub(t, clk)
	room, host, guest, aliceC, bobC := twoSeated(t, h)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	drain(aliceC)
	drain(bobC)

	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "submit", Move: "rock"})
	aliceMsgs := drain(aliceC)
	bobMsgs := drain(bobC)
	av := viewOf(lastType(aliceMsgs, "view"))
	if av == nil || av["yourMove"] != "rock" {
		t.Fatalf("alice should see her move: %+v", av)
	}
	if lastType(bobMsgs, "view") != nil {
		t.Fatalf("bob should not get a view on alice submit: %+v", bobMsgs)
	}

	h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "submit", Move: "scissors"})
	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "ready"})
	bobMsgs = drain(bobC)
	bv := viewOf(lastType(bobMsgs, "view"))
	if bv != nil && bv["opponentReady"] != true {
		t.Fatalf("bob should see opponent ready, not the card: %+v", bv)
	}
	if bv != nil && bv["yourMove"] == "rock" {
		t.Fatal("bob must not see alice's hidden rock")
	}

	h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "ready"})
	aliceMsgs = drain(aliceC)
	bobMsgs = drain(bobC)
	ar := lastType(aliceMsgs, "round.resolved")
	br := lastType(bobMsgs, "round.resolved")
	if ar == nil || br == nil {
		t.Fatalf("expected round.resolved alice=%v bob=%v", ar, br)
	}
	ares, _ := ar["result"].(map[string]any)
	bres, _ := br["result"].(map[string]any)
	if ares["yourMove"] != "rock" || ares["opponentMove"] != "scissors" || ares["youWon"] != true {
		t.Fatalf("alice result %+v", ares)
	}
	if bres["yourMove"] != "scissors" || bres["opponentMove"] != "rock" || bres["youWon"] != false {
		t.Fatalf("bob result %+v", bres)
	}

	s := h.get(room.ID)
	s.mu.Lock()
	mid := s.matchID
	hist := s.match.History()
	s.mu.Unlock()
	got, ok := st.Match(mid)
	if !ok || len(got.Rounds) != 1 {
		t.Fatalf("persisted %+v ok=%v", got, ok)
	}
	if hist[0].Winner != game.PlayerID(host.ID) {
		t.Fatalf("winner %+v", hist[0])
	}
}

func TestTimeoutNoMoveIsDrawAndLateTimerIgnored(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, _ := playHubPick(t, clk, func() game.Move { return game.MoveRock })
	room, host, guest, aliceC, bobC := twoSeated(t, h)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	drain(aliceC)
	drain(bobC)

	if n := clk.Fire(5 * time.Second); n != 1 {
		t.Fatalf("timeout jobs fired=%d", n)
	}
	aliceMsgs := drain(aliceC)
	res := lastType(aliceMsgs, "round.resolved")
	if res == nil {
		t.Fatalf("want round.resolved, got %+v", aliceMsgs)
	}
	body, _ := res["result"].(map[string]any)
	if body["draw"] != true {
		t.Fatalf("double auto-pick of rock should draw: %+v", body)
	}
	if body["yourAutoPicked"] != true || body["opponentAutoPicked"] != true {
		t.Fatalf("both should be auto-picked: %+v", body)
	}
	if body["yourMove"] != "rock" || body["opponentMove"] != "rock" {
		t.Fatalf("moves %+v", body)
	}

	// Next round is 2 (first-to-2). A late round-1 timer must be a no-op.
	s := h.get(room.ID)
	s.mu.Lock()
	if s.match.RoundNumber() != 2 {
		s.mu.Unlock()
		t.Fatalf("round=%d", s.match.RoundNumber())
	}
	before := len(s.match.History())
	s.mu.Unlock()
	h.turnTimeout(room.ID, 1)
	s.mu.Lock()
	after := len(s.match.History())
	round := s.match.RoundNumber()
	s.mu.Unlock()
	if after != before || round != 2 {
		t.Fatalf("stale timeout mutated state hist %d→%d round=%d", before, after, round)
	}

	// Play round 2 for real. Result-pause then round-1 timeout still ignored.
	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "submit", Move: "paper"})
	h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "submit", Move: "rock"})
	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "ready"})
	h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "ready"})
	h.turnTimeout(room.ID, 1)
	s.mu.Lock()
	// first-to-2: draw then paper-beats-rock → 1–0, still playing. Late
	// round-1 timer must not invent a third round or finish the match.
	if s.match.Status() != game.StatusPlaying {
		t.Fatalf("status=%s", s.match.Status())
	}
	if len(s.match.History()) != 2 {
		t.Fatalf("history=%d", len(s.match.History()))
	}
	if s.match.RoundNumber() != 3 {
		t.Fatalf("round=%d", s.match.RoundNumber())
	}
	s.mu.Unlock()
	_ = bobC
}

func TestWinsToFinishPersistsStats(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, st := playHub(t, clk)
	ctx := context.Background()
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
	room, err := h.Create(ctx, host, 1, 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Join(ctx, guest, room.Passcode); err != nil {
		t.Fatal(err)
	}
	s := h.get(room.ID)
	_ = seat(h, s, host)
	_ = seat(h, s, guest)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "submit", Move: "scissors"})
	h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "submit", Move: "paper"})
	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "ready"})
	h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "ready"})

	s.mu.Lock()
	if s.match.Status() != game.StatusFinished {
		s.mu.Unlock()
		t.Fatal("match should be over")
	}
	mid := s.matchID
	s.mu.Unlock()

	got, ok := st.Match(mid)
	if !ok || !got.Ended || got.WinnerID != host.ID || len(got.Rounds) != 1 {
		t.Fatalf("match %+v ok=%v", got, ok)
	}
	hs := st.Stats(host.ID)
	gs := st.Stats(guest.ID)
	if hs.Played != 1 || hs.Wins != 1 || hs.Losses != 0 || hs.Scissors != 1 {
		t.Fatalf("host stats %+v", hs)
	}
	if gs.Played != 1 || gs.Wins != 0 || gs.Losses != 1 || gs.Paper != 1 {
		t.Fatalf("guest stats %+v", gs)
	}

	clk.Fire(3 * time.Second) // match.finished after the result pause
}

func TestForfeitOnGraceDuringMatch(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, st := playHub(t, clk)
	room, host, guest, aliceC, bobC := twoSeated(t, h)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	drain(aliceC)
	drain(bobC)

	h.detach(bobC)
	aliceMsgs := drain(aliceC)
	if lastType(aliceMsgs, "opponent.disconnected") == nil {
		t.Fatalf("want opponent.disconnected, got %+v", aliceMsgs)
	}

	if n := clk.Fire(50 * time.Millisecond); n < 1 {
		t.Fatalf("grace jobs=%d", n)
	}
	aliceMsgs = drain(aliceC)
	fin := lastType(aliceMsgs, "match.finished")
	if fin == nil {
		t.Fatalf("want match.finished after grace, got %+v", aliceMsgs)
	}
	v := viewOf(fin)
	if v["winnerId"] != host.ID {
		t.Fatalf("winner %+v", v)
	}
	s := h.get(room.ID)
	s.mu.Lock()
	mid := s.matchID
	stillThere := false
	_, stillThere = s.findLocked(guest.ID)
	s.mu.Unlock()
	if !stillThere {
		t.Fatal("forfeit should keep the seat so they can rematch")
	}
	got, _ := st.Match(mid)
	if got.WinnerID != host.ID {
		t.Fatalf("persisted winner %s", got.WinnerID)
	}
}

func TestClaimWinWhileDisconnected(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, _ := playHub(t, clk)
	room, host, _, aliceC, bobC := twoSeated(t, h)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	drain(aliceC)
	h.detach(bobC)
	drain(aliceC)
	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "claim_win"})
	msgs := drain(aliceC)
	if lastType(msgs, "match.finished") == nil {
		t.Fatalf("claim_win should finish: %+v", msgs)
	}
}

func TestPlayAgainStartsNewMatch(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, st := playHub(t, clk)
	ctx := context.Background()
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
	room, _ := h.Create(ctx, host, 1, 8)
	_, _ = h.Join(ctx, guest, room.Passcode)
	s := h.get(room.ID)
	aliceC := seat(h, s, host)
	bobC := seat(h, s, guest)
	_ = h.StartMatch(room.ID, host.ID)
	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "submit", Move: "rock"})
	h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "submit", Move: "scissors"})
	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "ready"})
	h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "ready"})
	drain(aliceC)
	drain(bobC)

	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "play_again"})
	msgs := drain(aliceC)
	wait := lastType(msgs, "play_again.waiting")
	if wait == nil || wait["ready"] != float64(1) {
		t.Fatalf("waiting %+v", wait)
	}
	h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "play_again"})
	msgs = drain(aliceC)
	started := lastType(msgs, "match.started")
	if started == nil {
		t.Fatalf("rematch started: %+v", msgs)
	}
	s.mu.Lock()
	if s.match == nil || s.match.Status() != game.StatusPlaying || s.match.RoundNumber() != 1 {
		t.Fatalf("new match %+v", s.match)
	}
	second := s.matchID
	s.mu.Unlock()
	if _, ok := st.Match(second); !ok {
		t.Fatal("second match not persisted")
	}
}

func TestConcurrentSubmitReadyRace(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, _ := playHub(t, clk)
	room, host, guest, _, _ := twoSeated(t, h)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		h.handlePlay(room.ID, host.ID, ClientMessage{Type: "submit", Move: "rock"})
		h.handlePlay(room.ID, host.ID, ClientMessage{Type: "ready"})
	}()
	go func() {
		defer wg.Done()
		h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "submit", Move: "paper"})
		h.handlePlay(room.ID, guest.ID, ClientMessage{Type: "ready"})
	}()
	wg.Wait()

	s := h.get(room.ID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.match == nil {
		t.Fatal("nil match")
	}
	if len(s.match.History()) != 1 {
		t.Fatalf("history=%d", len(s.match.History()))
	}
	rr := s.match.History()[0]
	if rr.Draw || rr.Winner != game.PlayerID(guest.ID) {
		t.Fatalf("paper beats rock: %+v", rr)
	}
}

func TestLeaveMidMatchForfeits(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, st := playHub(t, clk)
	room, host, guest, aliceC, _ := twoSeated(t, h)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	drain(aliceC)
	if _, err := h.Leave(context.Background(), room.ID, guest.ID); err != nil {
		t.Fatal(err)
	}
	msgs := drain(aliceC)
	if lastType(msgs, "match.finished") == nil {
		t.Fatalf("leave should forfeit: %+v", msgs)
	}
	s := h.get(room.ID)
	s.mu.Lock()
	mid := s.matchID
	s.mu.Unlock()
	got, _ := st.Match(mid)
	if got.WinnerID != host.ID {
		t.Fatalf("winner %s", got.WinnerID)
	}
}

func TestTimeoutUsesSubmittedNotReadyMove(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, st := playHubPick(t, clk, func() game.Move { return game.MoveScissors })
	room, host, _, aliceC, bobC := twoSeated(t, h)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	drain(aliceC)
	drain(bobC)
	h.handlePlay(room.ID, host.ID, ClientMessage{Type: "submit", Move: "rock"})
	clk.Fire(5 * time.Second)
	msgs := drain(aliceC)
	res := lastType(msgs, "round.resolved")
	body, _ := res["result"].(map[string]any)
	if body["yourMove"] != "rock" || body["opponentMove"] != "scissors" || body["youWon"] != true {
		t.Fatalf("%+v", body)
	}
	if body["yourAutoPicked"] != false || body["opponentAutoPicked"] != true {
		t.Fatalf("auto flags %+v", body)
	}
	s := h.get(room.ID)
	s.mu.Lock()
	mid := s.matchID
	s.mu.Unlock()
	got, ok := st.Match(mid)
	if !ok || len(got.Rounds) != 1 {
		t.Fatalf("persisted %+v ok=%v", got, ok)
	}
	auto := 0
	for _, v := range got.Rounds[0].AutoPicked {
		if v {
			auto++
		}
	}
	if auto != 1 {
		t.Fatalf("want 1 auto-picked move, got %+v", got.Rounds[0].AutoPicked)
	}
}

func playHubPick(t *testing.T, clk *fakeClock, pick game.Pick) (*Hub, *rooms.Memory) {
	t.Helper()
	st := rooms.NewMemory()
	h := New(st, Config{
		Grace:       50 * time.Millisecond,
		After:       clk.After,
		ResultPause: 3 * time.Second,
		Pick:        pick,
	})
	return h, st
}

func TestLateSocketGetsMatchStarted(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, _ := playHub(t, clk)
	room, host, _, aliceC, bobC := twoSeated(t, h)
	h.detach(aliceC)
	h.detach(bobC)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	s := h.get(room.ID)
	c := h.attach(s, host, nil)
	if c == nil {
		t.Fatal("attach")
	}
	msgs := drain(c)
	started := lastType(msgs, "match.started")
	if started == nil {
		t.Fatalf("late socket should get match.started, got %+v", msgs)
	}
	v := viewOf(started)
	if v["roundNumber"] != float64(1) {
		t.Fatalf("snapshot %+v", started)
	}
}

func TestReconnectResumesClock(t *testing.T) {
	t.Parallel()
	clk := &fakeClock{}
	h, _ := playHub(t, clk)
	room, host, guest, aliceC, bobC := twoSeated(t, h)
	if err := h.StartMatch(room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	drain(aliceC)
	h.detach(bobC)
	s := h.get(room.ID)
	s.mu.Lock()
	if !s.paused {
		s.mu.Unlock()
		t.Fatal("clock should pause")
	}
	s.mu.Unlock()

	c2 := h.attach(s, guest, nil)
	if c2 == nil {
		t.Fatal("attach")
	}
	s.mu.Lock()
	paused := s.paused
	s.mu.Unlock()
	if paused {
		t.Fatal("clock should resume")
	}
}
