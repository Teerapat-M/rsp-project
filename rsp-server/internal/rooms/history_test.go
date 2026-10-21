package rooms

import (
	"context"
	"testing"

	"github.com/Teerapat-M/rsp-project/internal/ident"
)

func TestMemoryHistoryAndStats(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	m := NewMemory()
	a, err := m.EnsureGuest(ctx, ident.Player{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", DisplayName: "Alice", Avatar: "rock", Color: "#111111"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.EnsureGuest(ctx, ident.Player{ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", DisplayName: "Bob", Avatar: "paper", Color: "#222222"})
	if err != nil {
		t.Fatal(err)
	}
	room, err := m.InsertRoom(ctx, a.ID, "123456", 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.AddPlayer(ctx, room.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	mid, err := m.InsertMatch(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SaveRound(ctx, mid, 1, a.ID, map[string]string{a.ID: "rock", b.ID: "scissors"}); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveRound(ctx, mid, 2, a.ID, map[string]string{a.ID: "paper", b.ID: "rock"}); err != nil {
		t.Fatal(err)
	}
	if err := m.FinishMatch(ctx, mid, a.ID); err != nil {
		t.Fatal(err)
	}
	_ = m.ApplyMatchStats(ctx, a.ID, true, false, 1, 1, 0)
	_ = m.ApplyMatchStats(ctx, b.ID, false, true, 1, 0, 1)

	st, err := m.UserStats(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if st.MatchesPlayed != 1 || st.Wins != 1 || st.WinRate != 100 || st.BestStreak != 1 {
		t.Fatalf("alice stats %+v", st)
	}
	if st.PickWins["rock"] != 1 || st.PickWins["paper"] != 1 {
		t.Fatalf("pick wins %+v", st.PickWins)
	}

	page, err := m.ListHistory(ctx, a.ID, 20, 0)
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("history %+v %v", page, err)
	}
	item := page.Items[0]
	if item.OpponentName != "Bob" || item.YouScore != 2 || item.OpponentScore != 0 || !item.Won {
		t.Fatalf("item %+v", item)
	}

	d, err := m.MatchDetail(ctx, a.ID, mid)
	if err != nil || len(d.Rounds) != 2 || d.Rounds[0].Result != "won" || d.Rounds[0].You != "rock" {
		t.Fatalf("detail %+v %v", d, err)
	}
	if _, err := m.MatchDetail(ctx, "cccccccc-cccc-cccc-cccc-cccccccccccc", mid); err != ErrNotFound {
		t.Fatalf("stranger: %v", err)
	}
}

func TestStreak(t *testing.T) {
	t.Parallel()
	if n := Streak([]string{"a", "a", "b", "a", "a", "a"}, "a"); n != 3 {
		t.Fatalf("got %d", n)
	}
	if n := Streak(nil, "a"); n != 0 {
		t.Fatalf("empty %d", n)
	}
}
