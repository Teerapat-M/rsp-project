package hub

import (
	"context"
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
