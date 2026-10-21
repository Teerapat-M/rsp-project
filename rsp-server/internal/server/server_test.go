package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/hub"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	h := hub.New(rooms.NewMemory(), hub.Config{Grace: 200 * time.Millisecond})
	return httptest.NewServer(New(h).Router())
}

func doJSON(t *testing.T, ts *httptest.Server, method, path, player string, body any) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.URL+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if player != "" {
		req.Header.Set("X-Player-ID", player)
		req.Header.Set("X-Player-Name", player)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func decode(t *testing.T, res *http.Response, dest any) {
	t.Helper()
	defer res.Body.Close()
	if dest == nil {
		_, _ = io.Copy(io.Discard, res.Body)
		return
	}
	if err := json.NewDecoder(res.Body).Decode(dest); err != nil {
		t.Fatal(err)
	}
}

func TestRESTCreateJoinLeave(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	res := doJSON(t, ts, http.MethodPost, "/rooms", "alice", map[string]int{"winsToFinish": 3, "turnSeconds": 15})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status %d", res.StatusCode)
	}
	var room rooms.Room
	decode(t, res, &room)
	if len(room.Passcode) != 6 || room.WinsToFinish != 3 {
		t.Fatalf("room %+v", room)
	}

	res = doJSON(t, ts, http.MethodPost, "/rooms/join", "bob", map[string]string{"passcode": room.Passcode})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("join status %d", res.StatusCode)
	}
	decode(t, res, &room)
	if len(room.Players) != 2 {
		t.Fatalf("players %d", len(room.Players))
	}

	res = doJSON(t, ts, http.MethodGet, "/rooms/"+room.ID, "alice", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("get %d", res.StatusCode)
	}
	decode(t, res, &room)

	res = doJSON(t, ts, http.MethodPost, "/rooms/"+room.ID+"/leave", "bob", nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("leave %d", res.StatusCode)
	}
	decode(t, res, nil)
}

func TestJoinWrongPasscodeAndFull(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	res := doJSON(t, ts, http.MethodPost, "/rooms", "alice", map[string]int{"winsToFinish": 1, "turnSeconds": 10})
	var room rooms.Room
	decode(t, res, &room)

	res = doJSON(t, ts, http.MethodPost, "/rooms/join", "bob", map[string]string{"passcode": "000000"})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("wrong status %d", res.StatusCode)
	}
	var errBody apiError
	decode(t, res, &errBody)
	if errBody.Code != "WRONG_PASSCODE" || errBody.Passcode != "000000" {
		t.Fatalf("wrong body %+v", errBody)
	}

	res = doJSON(t, ts, http.MethodPost, "/rooms/join", "bob", map[string]string{"passcode": room.Passcode})
	decode(t, res, &room)

	res = doJSON(t, ts, http.MethodPost, "/rooms/join", "carol", map[string]string{"passcode": room.Passcode})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("full status %d", res.StatusCode)
	}
	decode(t, res, &errBody)
	if errBody.Code != "ROOM_FULL" || errBody.Passcode != room.Passcode {
		t.Fatalf("full body %+v", errBody)
	}
}

func TestDevGuestBearerCreatesRoom(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	res, err := http.Post(ts.URL+"/dev/guest", "application/json", bytes.NewBufferString(`{"displayName":"Mina","avatar":"paper"}`))
	if err != nil {
		t.Fatal(err)
	}
	var session struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			IsGuest     bool   `json:"isGuest"`
		} `json:"user"`
	}
	decode(t, res, &session)
	if session.AccessToken == "" || session.User.DisplayName != "Mina" || !session.User.IsGuest {
		t.Fatalf("%+v", session)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/rooms", bytes.NewBufferString(`{"winsToFinish":2,"turnSeconds":10}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d", res.StatusCode)
	}
	var room rooms.Room
	decode(t, res, &room)
	if room.HostID != session.User.ID {
		t.Fatalf("host %s want %s", room.HostID, session.User.ID)
	}
}

func TestUnauthorized(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()
	res := doJSON(t, ts, http.MethodPost, "/rooms", "", map[string]int{"winsToFinish": 1, "turnSeconds": 10})
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", res.StatusCode)
	}
	var errBody apiError
	decode(t, res, &errBody)
	if errBody.Code != "UNAUTHORIZED" {
		t.Fatalf("%+v", errBody)
	}
}

func TestStartIsPhase3Hook(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()
	res := doJSON(t, ts, http.MethodPost, "/rooms", "alice", map[string]int{"winsToFinish": 1, "turnSeconds": 10})
	var room rooms.Room
	decode(t, res, &room)
	res = doJSON(t, ts, http.MethodPost, "/rooms/"+room.ID+"/start", "alice", nil)
	if res.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status %d", res.StatusCode)
	}
	var errBody apiError
	decode(t, res, &errBody)
	if errBody.Code != "NOT_IMPLEMENTED" {
		t.Fatalf("%+v", errBody)
	}
}

func TestWSJoinLeavePresence(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	res := doJSON(t, ts, http.MethodPost, "/rooms", "alice", map[string]int{"winsToFinish": 3, "turnSeconds": 15})
	var room rooms.Room
	decode(t, res, &room)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room=" + room.ID + "&token=alice"
	aliceWS, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer aliceWS.Close()
	sawRoom := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_ = aliceWS.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		var msg map[string]any
		if err := aliceWS.ReadJSON(&msg); err != nil {
			continue
		}
		if msg["type"] == "room.updated" {
			sawRoom = true
			break
		}
	}
	if !sawRoom {
		t.Fatal("alice did not get room.updated on connect")
	}

	res = doJSON(t, ts, http.MethodPost, "/rooms/join", "bob", map[string]string{"passcode": room.Passcode})
	decode(t, res, &room)

	deadline = time.Now().Add(2 * time.Second)
	sawJoin := false
	for time.Now().Before(deadline) {
		_ = aliceWS.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		var msg map[string]any
		if err := aliceWS.ReadJSON(&msg); err != nil {
			continue
		}
		if msg["type"] == "player.joined" || msg["type"] == "room.updated" {
			if msg["type"] == "room.updated" {
				raw, _ := json.Marshal(msg["room"])
				var r rooms.Room
				_ = json.Unmarshal(raw, &r)
				if len(r.Players) == 2 {
					sawJoin = true
					break
				}
			}
			if msg["type"] == "player.joined" {
				sawJoin = true
				break
			}
		}
	}
	if !sawJoin {
		t.Fatal("alice never saw bob join")
	}

	bobURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room=" + room.ID + "&token=bob"
	bobWS, _, err := websocket.DefaultDialer.Dial(bobURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer bobWS.Close()

	if err := bobWS.WriteJSON(map[string]string{"type": "lobby.ready"}); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(2 * time.Second)
	sawReady := false
	for time.Now().Before(deadline) {
		_ = aliceWS.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		var msg map[string]any
		if err := aliceWS.ReadJSON(&msg); err != nil {
			continue
		}
		if msg["type"] == "lobby.ready" {
			sawReady = true
			break
		}
		if msg["type"] == "room.updated" {
			raw, _ := json.Marshal(msg["room"])
			var r rooms.Room
			_ = json.Unmarshal(raw, &r)
			for _, p := range r.Players {
				if p.Ready {
					sawReady = true
				}
			}
			if sawReady {
				break
			}
		}
	}
	if !sawReady {
		t.Fatal("did not see lobby ready")
	}

	_ = bobWS.WriteJSON(map[string]string{"type": "leave"})
	deadline = time.Now().Add(2 * time.Second)
	sawLeft := false
	for time.Now().Before(deadline) {
		_ = aliceWS.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		var msg map[string]any
		if err := aliceWS.ReadJSON(&msg); err != nil {
			continue
		}
		if msg["type"] == "player.left" {
			sawLeft = true
			break
		}
		if msg["type"] == "room.updated" {
			raw, _ := json.Marshal(msg["room"])
			var r rooms.Room
			_ = json.Unmarshal(raw, &r)
			if len(r.Players) == 1 {
				sawLeft = true
				break
			}
		}
	}
	if !sawLeft {
		t.Fatal("alice never saw bob leave")
	}
}

func TestHealth(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()
	res, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["phase"] != float64(2) {
		t.Fatalf("%v", body)
	}
}
