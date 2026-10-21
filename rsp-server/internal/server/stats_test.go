package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestStatsAndHistoryAfterMatch(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()
	alice := doJSON(t, ts, http.MethodPost, "/rooms", "alice", map[string]int{"winsToFinish": 1, "turnSeconds": 8})
	var room struct {
		ID       string `json:"id"`
		Passcode string `json:"passcode"`
	}
	decode(t, alice, &room)
	bob := doJSON(t, ts, http.MethodPost, "/rooms/join", "bob", map[string]string{"passcode": room.Passcode})
	if bob.StatusCode != http.StatusOK {
		t.Fatalf("join %d", bob.StatusCode)
	}
	decode(t, bob, nil)

	start := doJSON(t, ts, http.MethodPost, "/rooms/"+room.ID+"/start", "alice", nil)
	if start.StatusCode != http.StatusNoContent {
		t.Fatalf("start %d", start.StatusCode)
	}
	decode(t, start, nil)

	// Empty stats before anyone has finished a match.
	res := doJSON(t, ts, http.MethodGet, "/stats", "alice", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("stats %d", res.StatusCode)
	}
	var st map[string]any
	decode(t, res, &st)
	if st["matchesPlayed"] != float64(0) {
		t.Fatalf("expected empty stats, got %+v", st)
	}

	res = doJSON(t, ts, http.MethodGet, "/matches", "alice", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("history %d", res.StatusCode)
	}
	var page struct {
		Items []json.RawMessage `json:"items"`
		Total int               `json:"total"`
	}
	decode(t, res, &page)
	if page.Total != 0 || page.Items == nil {
		t.Fatalf("empty history %+v", page)
	}
}
