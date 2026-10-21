package rooms

import (
	"context"
	"errors"
	"testing"

	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/passcode"
)

func TestCheckJoin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		members   []string
		candidate string
		want      error
	}{
		{name: "empty room", candidate: "a"},
		{name: "second seat", members: []string{"a"}, candidate: "b"},
		{name: "already in", members: []string{"a", "b"}, candidate: "a"},
		{name: "full", members: []string{"a", "b"}, candidate: "c", want: ErrRoomFull},
		{name: "full third", members: []string{"host", "guest"}, candidate: "late", want: ErrRoomFull},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := CheckJoin(tt.members, tt.candidate)
			if !errors.Is(err, tt.want) {
				t.Fatalf("CheckJoin(%v, %q) = %v, want %v", tt.members, tt.candidate, err, tt.want)
			}
		})
	}
}

func TestValidateSettings(t *testing.T) {
	t.Parallel()
	if err := ValidateSettings(1, 1); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSettings(0, 10); !errors.Is(err, ErrInvalidWins) {
		t.Fatalf("wins: %v", err)
	}
	if err := ValidateSettings(3, 0); !errors.Is(err, ErrInvalidTurn) {
		t.Fatalf("turn: %v", err)
	}
}

func TestNormalizePasscode(t *testing.T) {
	t.Parallel()
	if g := NormalizePasscode("123 456"); g != "123456" {
		t.Fatalf("got %q", g)
	}
	if g := NormalizePasscode("12-34-56"); g != "123456" {
		t.Fatalf("got %q", g)
	}
}

func TestMemoryPasscodeUniqueness(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := NewMemory()
	host := ident.Player{ID: "11111111-1111-1111-1111-111111111111", DisplayName: "Host"}
	if _, err := st.EnsureGuest(ctx, host); err != nil {
		t.Fatal(err)
	}

	n := 0
	gen := func() string {
		n++
		if n <= 2 {
			return "111111"
		}
		return "222222"
	}
	taken := func(code string) bool {
		_, err := st.GetOpenByPasscode(ctx, code)
		return err == nil
	}

	first, err := st.InsertRoom(ctx, host.ID, "111111", 3, 15)
	if err != nil {
		t.Fatal(err)
	}
	if first.Passcode != "111111" {
		t.Fatalf("first passcode %s", first.Passcode)
	}

	code, err := passcode.Allocate(taken, gen, 10)
	if err != nil {
		t.Fatal(err)
	}
	if code != "222222" {
		t.Fatalf("allocated %q after collision, want 222222", code)
	}
	if _, err := st.InsertRoom(ctx, host.ID, "111111", 3, 15); !errors.Is(err, ErrPasscodeTaken) {
		t.Fatalf("duplicate insert err = %v, want ErrPasscodeTaken", err)
	}

	closed := first
	if err := st.CloseRoom(ctx, closed.ID); err != nil {
		t.Fatal(err)
	}
	// After close the code may be reused (partial unique index).
	if _, err := st.InsertRoom(ctx, host.ID, "111111", 3, 15); err != nil {
		t.Fatalf("reuse after close: %v", err)
	}
}

func TestMemoryWrongPasscodeAndFull(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := NewMemory()

	host := mustGuest(t, st, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "Host")
	p2 := mustGuest(t, st, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "P2")
	p3 := mustGuest(t, st, "cccccccc-cccc-cccc-cccc-cccccccccccc", "P3")

	room, err := st.InsertRoom(ctx, host.ID, "424242", 3, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.AddPlayer(ctx, room.ID, host.ID); err != nil {
		t.Fatal(err)
	}

	_, err = st.GetOpenByPasscode(ctx, "000000")
	if !errors.Is(err, ErrWrongPasscode) {
		t.Fatalf("missing room: %v", err)
	}

	open, err := st.GetOpenByPasscode(ctx, "424242")
	if err != nil {
		t.Fatal(err)
	}
	members, err := st.ListPlayers(ctx, open.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckJoin(MemberIDs(members), p2.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.AddPlayer(ctx, open.ID, p2.ID); err != nil {
		t.Fatal(err)
	}

	members, _ = st.ListPlayers(ctx, open.ID)
	if err := CheckJoin(MemberIDs(members), p3.ID); !errors.Is(err, ErrRoomFull) {
		t.Fatalf("third join: %v, want full", err)
	}
	// Idempotent: p2 can "join" again.
	if err := CheckJoin(MemberIDs(members), p2.ID); err != nil {
		t.Fatalf("rejoin: %v", err)
	}
}

func mustGuest(t *testing.T, st Store, id, name string) ident.Player {
	t.Helper()
	p, err := st.EnsureGuest(context.Background(), ident.Player{ID: id, DisplayName: name})
	if err != nil {
		t.Fatal(err)
	}
	return p
}
