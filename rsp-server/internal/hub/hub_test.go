package hub

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
)

func testHub(t *testing.T, grace time.Duration) (*Hub, *rooms.Memory) {
	t.Helper()
	st := rooms.NewMemory()
	h := New(st, Config{Grace: grace})
	return h, st
}

func player(id, name string) ident.Player {
	return ident.Player{ID: id, DisplayName: name, Avatar: ident.DefaultAvatar, Color: ident.DefaultColor}
}

func TestCreateJoinLeave(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h, _ := testHub(t, time.Second)
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
	late := player("cccccccc-cccc-cccc-cccc-cccccccccccc", "Late")

	room, err := h.Create(ctx, host, 3, 15)
	if err != nil {
		t.Fatal(err)
	}
	if len(room.Passcode) != 6 {
		t.Fatalf("passcode %q", room.Passcode)
	}
	if len(room.Players) != 1 || !room.Players[0].IsHost {
		t.Fatalf("host not seated: %+v", room.Players)
	}

	joined, err := h.Join(ctx, guest, room.Passcode)
	if err != nil {
		t.Fatal(err)
	}
	if len(joined.Players) != 2 {
		t.Fatalf("want 2 players, got %d", len(joined.Players))
	}

	_, err = h.Join(ctx, late, room.Passcode)
	if !errors.Is(err, rooms.ErrRoomFull) {
		t.Fatalf("third: %v", err)
	}

	again, err := h.Join(ctx, guest, room.Passcode)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Players) != 2 {
		t.Fatalf("idempotent join grew the room: %d", len(again.Players))
	}

	_, err = h.Join(ctx, guest, "000000")
	if !errors.Is(err, rooms.ErrWrongPasscode) {
		t.Fatalf("wrong code: %v", err)
	}

	left, err := h.Leave(ctx, room.ID, guest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(left.Players) != 1 {
		t.Fatalf("after leave: %d", len(left.Players))
	}

	// Seat frees up.
	if _, err := h.Join(ctx, late, room.Passcode); err != nil {
		t.Fatal(err)
	}

	if _, err := h.Leave(ctx, room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	got, err := h.Get(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.HostID != late.ID {
		t.Fatalf("host should transfer, got %s", got.HostID)
	}

	if _, err := h.Leave(ctx, room.ID, late.ID); err != nil {
		t.Fatal(err)
	}
	if h.LiveCount() != 0 {
		t.Fatalf("empty room should be dropped, live=%d", h.LiveCount())
	}
	_, err = h.Join(ctx, host, room.Passcode)
	if !errors.Is(err, rooms.ErrWrongPasscode) {
		t.Fatalf("closed passcode should miss, got %v", err)
	}
}

func TestCreateRejectsBadSettings(t *testing.T) {
	t.Parallel()
	h, _ := testHub(t, time.Second)
	_, err := h.Create(context.Background(), player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "A"), 0, 10)
	if !errors.Is(err, rooms.ErrInvalidWins) {
		t.Fatal(err)
	}
}

func TestConcurrentJoinOneWinner(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h, _ := testHub(t, time.Second)
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	room, err := h.Create(ctx, host, 1, 10)
	if err != nil {
		t.Fatal(err)
	}

	const n = 24
	var ok atomic.Int32
	var full atomic.Int32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			p := player(
				ident.Normalize(string(rune('A'+i))).String(),
				"P",
			)
			// Unique UUIDs: hash the index.
			p.ID = ident.Normalize("racer-" + string(rune('a'+i))).String()
			_, err := h.Join(ctx, p, room.Passcode)
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, rooms.ErrRoomFull):
				full.Add(1)
			default:
				t.Errorf("join: %v", err)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("winners=%d want 1", ok.Load())
	}
	if full.Load() != int32(n-1) {
		t.Fatalf("full=%d want %d", full.Load(), n-1)
	}
	got, _ := h.Get(ctx, room.ID)
	if len(got.Players) != 2 {
		t.Fatalf("players=%d", len(got.Players))
	}
}

func TestLobbyReadyAndCleanup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h, _ := testHub(t, time.Second)
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
	room, err := h.Create(ctx, host, 3, 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Join(ctx, guest, room.Passcode); err != nil {
		t.Fatal(err)
	}
	snap, err := h.SetLobbyReady(room.ID, host.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Players[0].Ready {
		t.Fatalf("host should be ready: %+v", snap.Players)
	}
}

func TestDisconnectGraceThenLeave(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := rooms.NewMemory()

	var mu sync.Mutex
	var pending []func()
	h := New(st, Config{
		Grace: 50 * time.Millisecond,
		After: func(_ time.Duration, f func()) func() bool {
			mu.Lock()
			pending = append(pending, f)
			idx := len(pending) - 1
			mu.Unlock()
			stopped := atomic.Bool{}
			return func() bool {
				if stopped.Swap(true) {
					return false
				}
				mu.Lock()
				pending[idx] = nil
				mu.Unlock()
				return true
			}
		},
	})

	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
	room, err := h.Create(ctx, host, 3, 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Join(ctx, guest, room.Passcode); err != nil {
		t.Fatal(err)
	}

	s := h.get(room.ID)
	s.mu.Lock()
	m, ok := s.findLocked(guest.ID)
	if !ok {
		s.mu.Unlock()
		t.Fatal("guest missing")
	}
	// Pretend a socket existed and dropped.
	fake := &client{hub: h, sess: s, player: guest, send: make(chan []byte, 1)}
	m.conns[fake] = struct{}{}
	s.mu.Unlock()

	h.detach(fake)

	got, _ := h.Get(ctx, room.ID)
	if len(got.Players) != 2 {
		t.Fatalf("still in room during grace, got %d", len(got.Players))
	}
	var connected bool
	for _, p := range got.Players {
		if p.ID == guest.ID {
			connected = p.Connected
		}
	}
	if connected {
		t.Fatal("guest should be marked disconnected")
	}

	mu.Lock()
	fns := append([]func(){}, pending...)
	mu.Unlock()
	for _, f := range fns {
		if f != nil {
			f()
		}
	}

	got, _ = h.Get(ctx, room.ID)
	if len(got.Players) != 1 {
		t.Fatalf("after grace leave, players=%d", len(got.Players))
	}
}

func TestReconnectCancelsGrace(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := rooms.NewMemory()
	var mu sync.Mutex
	var pending []func()
	h := New(st, Config{
		Grace: time.Second,
		After: func(_ time.Duration, f func()) func() bool {
			mu.Lock()
			pending = append(pending, f)
			idx := len(pending) - 1
			mu.Unlock()
			var stopped atomic.Bool
			return func() bool {
				if stopped.Swap(true) {
					return false
				}
				mu.Lock()
				pending[idx] = nil
				mu.Unlock()
				return true
			}
		},
	})
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
	room, _ := h.Create(ctx, host, 3, 15)
	_, _ = h.Join(ctx, guest, room.Passcode)

	s := h.get(room.ID)
	s.mu.Lock()
	m, _ := s.findLocked(guest.ID)
	fake := &client{hub: h, sess: s, player: guest, send: make(chan []byte, 4)}
	m.conns[fake] = struct{}{}
	s.mu.Unlock()
	h.detach(fake)

	// Re-attach cancels grace.
	c2 := h.attach(s, guest, nil)
	if c2 == nil {
		t.Fatal("attach")
	}

	mu.Lock()
	var leftover int
	for _, f := range pending {
		if f != nil {
			leftover++
		}
	}
	mu.Unlock()
	if leftover != 0 {
		t.Fatalf("grace should be cancelled, leftover=%d", leftover)
	}
	got, _ := h.Get(ctx, room.ID)
	if len(got.Players) != 2 {
		t.Fatalf("players=%d", len(got.Players))
	}
}

func TestGetDoesNotAdvanceSeq(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h, _ := testHub(t, time.Second)
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
	room, err := h.Create(ctx, host, 3, 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Join(ctx, guest, room.Passcode); err != nil {
		t.Fatal(err)
	}
	a, err := h.Get(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.Get(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Seq != b.Seq {
		t.Fatalf("GET bumped seq %d -> %d", a.Seq, b.Seq)
	}
	ready, err := h.SetLobbyReady(room.ID, host.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if ready.Seq <= a.Seq {
		t.Fatalf("ready seq %d want > %d", ready.Seq, a.Seq)
	}
	c, err := h.Get(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if c.Seq != ready.Seq {
		t.Fatalf("GET after ready seq=%d want %d", c.Seq, ready.Seq)
	}
}

func TestActiveRoomFor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h, _ := testHub(t, time.Second)
	host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")

	if _, err := h.ActiveRoomFor(host.ID); !errors.Is(err, rooms.ErrNotFound) {
		t.Fatalf("empty: %v", err)
	}

	room, err := h.Create(ctx, host, 3, 15)
	if err != nil {
		t.Fatal(err)
	}
	got, err := h.ActiveRoomFor(host.ID)
	if err != nil || got.ID != room.ID || got.Status != rooms.StatusWaiting {
		t.Fatalf("waiting: %+v %v", got, err)
	}
	if _, err := h.Join(ctx, guest, room.Passcode); err != nil {
		t.Fatal(err)
	}
	g, err := h.ActiveRoomFor(guest.ID)
	if err != nil || g.ID != room.ID {
		t.Fatalf("guest: %+v %v", g, err)
	}
	if _, err := h.Leave(ctx, room.ID, host.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ActiveRoomFor(host.ID); !errors.Is(err, rooms.ErrNotFound) {
		t.Fatalf("left host: %v", err)
	}
}

func TestConcurrentLobbyReadyDualSocketSeq(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for iter := 0; iter < 32; iter++ {
		h, _ := testHub(t, time.Second)
		host := player("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
		guest := player("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Guest")
		room, err := h.Create(ctx, host, 3, 15)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := h.Join(ctx, guest, room.Passcode); err != nil {
			t.Fatal(err)
		}
		s := h.get(room.ID)
		// Two sockets per player: waiting-room loadRoom used to close and
		// reopen WS, so both were in m.conns while Ready was clicked.
		hostOld := seat(h, s, host)
		guestOld := seat(h, s, guest)
		hostNew := seat(h, s, host)
		guestNew := seat(h, s, guest)
		drain(hostOld)
		drain(guestOld)
		drain(hostNew)
		drain(guestNew)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := h.SetLobbyReady(room.ID, host.ID, true); err != nil {
				t.Errorf("iter %d host ready: %v", iter, err)
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := h.SetLobbyReady(room.ID, guest.ID, true); err != nil {
				t.Errorf("iter %d guest ready: %v", iter, err)
			}
		}()
		wg.Wait()

		snap, err := h.Get(ctx, room.ID)
		if err != nil {
			t.Fatal(err)
		}
		if n := readyCount(snap); n != 2 {
			t.Fatalf("iter %d hub ready=%d seq=%d", iter, n, snap.Seq)
		}

		hostOldMsgs := drain(hostOld)
		hostNewMsgs := drain(hostNew)
		guestOldMsgs := drain(guestOld)
		guestNewMsgs := drain(guestNew)
		for name, msgs := range map[string][]map[string]any{
			"hostOld": hostOldMsgs, "hostNew": hostNewMsgs,
			"guestOld": guestOldMsgs, "guestNew": guestNewMsgs,
		} {
			assertLobbyReadyOrder(t, iter, name, msgs)
			last := lastType(msgs, "room.updated")
			if last == nil {
				t.Fatalf("iter %d %s missing room.updated: %+v", iter, name, msgs)
			}
			got := roomFromMsg(last)
			if n := readyCount(got); n != 2 {
				t.Fatalf("iter %d %s last snapshot ready=%d seq=%d players=%+v", iter, name, n, got.Seq, got.Players)
			}
		}

		// Browser dual-socket race: new WS delivers 2/2, then the old WS
		// drains a 1/2 frame. Seq skip must keep 2/2.
		hostView := applyLobbyFrames(append(append([]map[string]any{}, hostNewMsgs...), hostOldMsgs...))
		guestView := applyLobbyFrames(append(append([]map[string]any{}, guestNewMsgs...), guestOldMsgs...))
		if n := readyCount(hostView); n != 2 {
			t.Fatalf("iter %d host dual-socket view ready=%d players=%+v", iter, n, hostView.Players)
		}
		if n := readyCount(guestView); n != 2 {
			t.Fatalf("iter %d guest dual-socket view ready=%d players=%+v", iter, n, guestView.Players)
		}
	}
}

func readyCount(r rooms.Room) int {
	n := 0
	for _, p := range r.Players {
		if p.Ready {
			n++
		}
	}
	return n
}

func roomFromMsg(m map[string]any) rooms.Room {
	raw, _ := json.Marshal(m["room"])
	var r rooms.Room
	_ = json.Unmarshal(raw, &r)
	return r
}

func jsonInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

func assertLobbyReadyOrder(t *testing.T, iter int, name string, msgs []map[string]any) {
	t.Helper()
	lastSnapSeq := int64(-1)
	sawPair := 0
	for _, m := range msgs {
		switch m["type"] {
		case "room.updated":
			lastSnapSeq = roomFromMsg(m).Seq
		case "lobby.ready":
			seq := jsonInt64(m["seq"])
			if seq != lastSnapSeq {
				t.Fatalf("iter %d %s lobby.ready seq=%d after room.updated seq=%d msgs=%+v", iter, name, seq, lastSnapSeq, msgs)
			}
			sawPair++
		}
	}
	if sawPair != 2 {
		t.Fatalf("iter %d %s want 2 lobby.ready pairs, got %d msgs=%+v", iter, name, sawPair, msgs)
	}
}

func applyLobbyFrames(msgs []map[string]any) rooms.Room {
	var cur rooms.Room
	var seq int64
	for _, m := range msgs {
		switch m["type"] {
		case "room.updated":
			r := roomFromMsg(m)
			if r.Seq > 0 && seq > 0 && r.Seq < seq {
				continue
			}
			cur = r
			if r.Seq > seq {
				seq = r.Seq
			}
		case "lobby.ready":
			evSeq := jsonInt64(m["seq"])
			if evSeq > 0 && seq > 0 && evSeq < seq {
				continue
			}
			pid, _ := m["playerId"].(string)
			ready, _ := m["playerReady"].(bool)
			for i := range cur.Players {
				if cur.Players[i].ID == pid {
					cur.Players[i].Ready = ready
				}
			}
			if evSeq > seq {
				seq = evSeq
				cur.Seq = evSeq
			}
		}
	}
	return cur
}
