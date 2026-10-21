package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGuestForbiddenOnStatsAndHistory(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	res, err := http.Post(ts.URL+"/auth/guest", "application/json",
		bytes.NewBufferString(`{"displayName":"Mina","avatar":"paper"}`))
	if err != nil {
		t.Fatal(err)
	}
	var sess struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			IsGuest bool `json:"isGuest"`
		} `json:"user"`
	}
	decode(t, res, &sess)
	if sess.AccessToken == "" || !sess.User.IsGuest {
		t.Fatalf("%+v", sess)
	}

	res = doBearer(t, ts, http.MethodGet, "/stats", sess.AccessToken, nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("guest jwt stats %d", res.StatusCode)
	}
	var errBody apiError
	decode(t, res, &errBody)
	if errBody.Code != "FORBIDDEN" {
		t.Fatalf("guest jwt stats %+v", errBody)
	}

	res = doBearer(t, ts, http.MethodGet, "/matches", sess.AccessToken, nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("guest jwt history %d", res.StatusCode)
	}
	decode(t, res, &errBody)
	if errBody.Code != "FORBIDDEN" {
		t.Fatalf("guest jwt history %+v", errBody)
	}

	res = doJSON(t, ts, http.MethodGet, "/stats", "alice", nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("dev guest stats %d", res.StatusCode)
	}
	decode(t, res, nil)
	res = doJSON(t, ts, http.MethodGet, "/matches", "alice", nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("dev guest history %d", res.StatusCode)
	}
	decode(t, res, nil)

	up := doBearer(t, ts, http.MethodPost, "/auth/upgrade", sess.AccessToken, map[string]string{
		"username": "mina_ok",
		"password": "secret1",
	})
	if up.StatusCode != http.StatusOK {
		t.Fatalf("upgrade %d", up.StatusCode)
	}
	var upgraded struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			IsGuest bool `json:"isGuest"`
		} `json:"user"`
	}
	decode(t, up, &upgraded)
	if upgraded.AccessToken == "" || upgraded.User.IsGuest {
		t.Fatalf("upgrade session %+v", upgraded)
	}
	res = doBearer(t, ts, http.MethodGet, "/stats", upgraded.AccessToken, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("upgraded stats %d", res.StatusCode)
	}
	decode(t, res, nil)
	res = doBearer(t, ts, http.MethodGet, "/matches", upgraded.AccessToken, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("upgraded history %d", res.StatusCode)
	}
	decode(t, res, nil)

	token := registerUser(t, ts, "bell_ok")
	res = doBearer(t, ts, http.MethodGet, "/stats", token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("registered stats %d", res.StatusCode)
	}
	decode(t, res, nil)
	res = doBearer(t, ts, http.MethodGet, "/matches", token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("registered history %d", res.StatusCode)
	}
	decode(t, res, nil)
}

func TestStatsAndHistoryAfterMatch(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()
	alice := registerUser(t, ts, "alice")
	bob := registerUser(t, ts, "bob")

	created := doBearer(t, ts, http.MethodPost, "/rooms", alice, map[string]int{"winsToFinish": 1, "turnSeconds": 8})
	var room struct {
		ID       string `json:"id"`
		Passcode string `json:"passcode"`
	}
	decode(t, created, &room)
	joined := doBearer(t, ts, http.MethodPost, "/rooms/join", bob, map[string]string{"passcode": room.Passcode})
	if joined.StatusCode != http.StatusOK {
		t.Fatalf("join %d", joined.StatusCode)
	}
	decode(t, joined, nil)

	start := doBearer(t, ts, http.MethodPost, "/rooms/"+room.ID+"/start", alice, nil)
	if start.StatusCode != http.StatusNoContent {
		t.Fatalf("start %d", start.StatusCode)
	}
	decode(t, start, nil)

	// Empty stats before anyone has finished a match.
	res := doBearer(t, ts, http.MethodGet, "/stats", alice, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("stats %d", res.StatusCode)
	}
	var st map[string]any
	decode(t, res, &st)
	if st["matchesPlayed"] != float64(0) {
		t.Fatalf("expected empty stats, got %+v", st)
	}

	res = doBearer(t, ts, http.MethodGet, "/matches", alice, nil)
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
