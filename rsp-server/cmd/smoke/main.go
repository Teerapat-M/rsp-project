// Smoke drives two HTTP+WS clients through a live server + Postgres.
// Run the server first (`make run`), then `make smoke`.
//
// Both sockets are opened before POST /rooms/:id/start. The hub only
// broadcasts match.started to currently attached clients; a late socket
// instead gets a snapshot (match.started if the match is already playing).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	base := getenv("SMOKE_BASE", "http://localhost:8080")
	if !strings.HasPrefix(base, "http") {
		base = "http://localhost" + base
	}
	wsBase := "ws" + strings.TrimPrefix(base, "http")

	health, err := http.Get(base + "/health")
	if err != nil {
		log.Fatalf("health: %v (is the server up? make run)", err)
	}
	body, _ := io.ReadAll(health.Body)
	_ = health.Body.Close()
	if health.StatusCode != 200 {
		log.Fatalf("health %d %s", health.StatusCode, body)
	}

	playSubmittedMatch(base, wsBase)
	playTimeoutRound(base, wsBase)
	playFirstToThreeWithDraw(base, wsBase)
	fmt.Println("smoke ok: first-to-1, timeout auto-pick, first-to-3 with draw, stats, and history against", base)
}

func playSubmittedMatch(base, wsBase string) {
	a, _, roomID, wa, wb := seatAndConnect(base, wsBase, 1, 15, "rock", "paper")
	defer wa.Close()
	defer wb.Close()

	mustStatus(http.StatusNoContent, postRaw(base+"/rooms/"+roomID+"/start", a.Access, nil))
	waitStarted(wa)
	waitStarted(wb)

	must(wa.WriteJSON(map[string]string{"type": "submit", "move": "rock"}))
	must(wb.WriteJSON(map[string]string{"type": "submit", "move": "scissors"}))
	must(wa.WriteJSON(map[string]string{"type": "ready"}))
	must(wb.WriteJSON(map[string]string{"type": "ready"}))
	fin := waitType(wa, "match.finished")
	view, _ := fin["view"].(map[string]any)
	if str(view["winnerId"]) != a.ID {
		log.Fatalf("want A to win, view=%v", view)
	}

	a = upgrade(base, a, "sma")
	stats := getJSON(base+"/stats", a.Access)
	if n, _ := stats["wins"].(float64); n < 1 {
		log.Fatalf("stats wins=%v", stats["wins"])
	}
	hist := getJSON(base+"/matches", a.Access)
	items, _ := hist["items"].([]any)
	if len(items) < 1 {
		log.Fatalf("history empty: %v", hist)
	}
	first, _ := items[0].(map[string]any)
	detail := getJSON(base+"/matches/"+str(first["id"]), a.Access)
	rounds, _ := detail["rounds"].([]any)
	if len(rounds) < 1 {
		log.Fatalf("match detail missing rounds: %v", detail)
	}
}

func playTimeoutRound(base, wsBase string) {
	a, _, roomID, wa, wb := seatAndConnect(base, wsBase, 1, 1, "rock", "paper")
	defer wa.Close()
	defer wb.Close()

	mustStatus(http.StatusNoContent, postRaw(base+"/rooms/"+roomID+"/start", a.Access, nil))
	waitStarted(wa)
	waitStarted(wb)

	var last map[string]any
	for i := 0; i < 8; i++ {
		msg := waitType(wa, "round.resolved")
		result, _ := msg["result"].(map[string]any)
		if !playable(result["yourMove"]) || !playable(result["opponentMove"]) {
			log.Fatalf("timeout round should auto-pick a throw, got %+v", result)
		}
		if result["yourAutoPicked"] != true || result["opponentAutoPicked"] != true {
			log.Fatalf("timeout round auto flags %+v", result)
		}
		last = result
		if result["matchOver"] == true {
			break
		}
	}
	if last["matchOver"] != true {
		log.Fatalf("timeout match did not finish after auto-picks: %+v", last)
	}
	waitType(wa, "match.finished")

	a = upgrade(base, a, "smb")
	hist := getJSON(base+"/matches", a.Access)
	items, _ := hist["items"].([]any)
	if len(items) < 1 {
		log.Fatalf("timeout history empty: %v", hist)
	}
	first, _ := items[0].(map[string]any)
	detail := getJSON(base+"/matches/"+str(first["id"]), a.Access)
	rounds, _ := detail["rounds"].([]any)
	if len(rounds) < 1 {
		log.Fatalf("timeout match detail missing rounds: %v", detail)
	}
	r0, _ := rounds[0].(map[string]any)
	if !playable(r0["you"]) || !playable(r0["opponent"]) {
		log.Fatalf("history timeout moves %+v", r0)
	}
	if r0["youAutoPicked"] != true || r0["opponentAutoPicked"] != true {
		log.Fatalf("history auto flags %+v", r0)
	}
}

func playFirstToThreeWithDraw(base, wsBase string) {
	a, b, roomID, wa, wb := seatAndConnect(base, wsBase, 3, 15, "paper", "scissors")
	defer wa.Close()
	defer wb.Close()

	mustStatus(http.StatusNoContent, postRaw(base+"/rooms/"+roomID+"/start", a.Access, nil))
	started := waitStarted(wa)
	waitStarted(wb)
	view, _ := started["view"].(map[string]any)
	you, _ := view["you"].(map[string]any)
	if str(you["avatar"]) != "paper" {
		log.Fatalf("self profile avatar on match.started: %v", you)
	}
	if str(view["opponentAvatar"]) != "scissors" {
		log.Fatalf("opponent profile avatar on match.started: %v", view)
	}

	playRound(wa, wb, "rock", "rock")
	draw := waitType(wa, "round.resolved")
	result, _ := draw["result"].(map[string]any)
	if result["draw"] != true || result["matchOver"] == true {
		log.Fatalf("round 1 should draw: %+v", result)
	}
	waitType(wa, "round.started")
	waitType(wb, "round.started")

	playRound(wa, wb, "rock", "scissors")
	r2 := waitType(wa, "round.resolved")
	result, _ = r2["result"].(map[string]any)
	if result["youWon"] != true || result["matchOver"] == true {
		log.Fatalf("round 2 A should win 1-0: %+v", result)
	}
	waitType(wa, "round.started")
	waitType(wb, "round.started")

	playRound(wa, wb, "scissors", "paper")
	r3 := waitType(wa, "round.resolved")
	result, _ = r3["result"].(map[string]any)
	if result["youWon"] != true || result["matchOver"] == true {
		log.Fatalf("round 3 A should win 2-0: %+v", result)
	}
	waitType(wa, "round.started")
	waitType(wb, "round.started")

	playRound(wa, wb, "paper", "rock")
	r4 := waitType(wa, "round.resolved")
	result, _ = r4["result"].(map[string]any)
	if result["youWon"] != true || result["matchOver"] != true {
		log.Fatalf("round 4 A should win the match 3-0: %+v", result)
	}
	fin := waitType(wa, "match.finished")
	finView, _ := fin["view"].(map[string]any)
	if str(finView["winnerId"]) != a.ID {
		log.Fatalf("first-to-3 winner %v want %s", finView["winnerId"], a.ID)
	}
	_ = b
}

func playRound(a, b *websocket.Conn, moveA, moveB string) {
	must(a.WriteJSON(map[string]string{"type": "submit", "move": moveA}))
	must(b.WriteJSON(map[string]string{"type": "submit", "move": moveB}))
	must(a.WriteJSON(map[string]string{"type": "ready"}))
	must(b.WriteJSON(map[string]string{"type": "ready"}))
}

func seatAndConnect(base, wsBase string, winsToFinish, turnSeconds int, avatarA, avatarB string) (session, session, string, *websocket.Conn, *websocket.Conn) {
	a := guest(base, "SmokeA", avatarA)
	b := guest(base, "SmokeB", avatarB)
	room := postJSON(base+"/rooms", a.Access, map[string]int{"winsToFinish": winsToFinish, "turnSeconds": turnSeconds})
	players, _ := room["players"].([]any)
	if len(players) > 0 {
		host, _ := players[0].(map[string]any)
		if str(host["avatar"]) != avatarA {
			log.Fatalf("create-room snapshot avatar=%v want %s", host["avatar"], avatarA)
		}
	}
	joined := postJSON(base+"/rooms/join", b.Access, map[string]string{"passcode": str(room["passcode"])})
	assertRoomAvatars(joined, a.ID, avatarA, b.ID, avatarB)
	roomID := str(room["id"])

	wa := dialWS(wsBase + "/ws?token=" + a.Access + "&room=" + roomID)
	wb := dialWS(wsBase + "/ws?token=" + b.Access + "&room=" + roomID)
	// Attach is complete once the connect snapshot arrives. Starting before
	// that races the hub broadcast of match.started.
	waitType(wa, "room.updated")
	waitType(wb, "room.updated")
	return a, b, roomID, wa, wb
}

func assertRoomAvatars(room map[string]any, idA, avatarA, idB, avatarB string) {
	players, _ := room["players"].([]any)
	got := map[string]string{}
	for _, raw := range players {
		p, _ := raw.(map[string]any)
		got[str(p["id"])] = str(p["avatar"])
	}
	if got[idA] != avatarA || got[idB] != avatarB {
		log.Fatalf("room snapshot avatars %v want %s=%s %s=%s", got, idA, avatarA, idB, avatarB)
	}
}

type session struct {
	Access string
	ID     string
}

func guest(base, name, avatar string) session {
	raw := postJSON(base+"/auth/guest", "", map[string]string{"displayName": name, "avatar": avatar, "color": "#9DB89A"})
	user, _ := raw["user"].(map[string]any)
	if str(user["avatar"]) != avatar {
		log.Fatalf("guest %s profile avatar=%v want %s", name, user["avatar"], avatar)
	}
	return session{Access: str(raw["accessToken"]), ID: str(user["id"])}
}

func upgrade(base string, s session, tag string) session {
	name := fmt.Sprintf("%s%d", tag, time.Now().UnixNano()%1e8)
	raw := postJSON(base+"/auth/upgrade", s.Access, map[string]string{
		"username": name,
		"password": "secret1",
	})
	user, _ := raw["user"].(map[string]any)
	if user["isGuest"] == true {
		log.Fatalf("upgrade still guest: %v", raw)
	}
	return session{Access: str(raw["accessToken"]), ID: str(user["id"])}
}

func dialWS(url string) *websocket.Conn {
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("ws %s: %v", url, err)
	}
	return c
}

func waitStarted(c *websocket.Conn) map[string]any {
	deadline := time.Now().Add(15 * time.Second)
	_ = c.SetReadDeadline(deadline)
	for {
		var msg map[string]any
		if err := c.ReadJSON(&msg); err != nil {
			log.Fatalf("wait match.started: %v", err)
		}
		typ := str(msg["type"])
		if typ == "match.started" {
			return msg
		}
		// Late / reconnect snapshot used to be a plain view; accept a playing
		// view so smoke still passes if the hub type ever regresses.
		if typ == "view" {
			view, _ := msg["view"].(map[string]any)
			if str(view["status"]) == "playing" {
				return msg
			}
		}
		if time.Now().After(deadline) {
			log.Fatalf("wait match.started: deadline")
		}
	}
}

func waitType(c *websocket.Conn, want string) map[string]any {
	_ = c.SetReadDeadline(time.Now().Add(15 * time.Second))
	for {
		var msg map[string]any
		if err := c.ReadJSON(&msg); err != nil {
			log.Fatalf("wait %s: %v", want, err)
		}
		if str(msg["type"]) == want {
			return msg
		}
	}
}

func playable(v any) bool {
	s := str(v)
	return s == "rock" || s == "paper" || s == "scissors"
}

func postJSON(url, token string, body any) map[string]any {
	res := postRaw(url, token, body)
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		log.Fatalf("POST %s -> %d %s", url, res.StatusCode, raw)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		log.Fatalf("decode %s: %v (%s)", url, err, raw)
	}
	return out
}

func postRaw(url, token string, body any) *http.Response {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(http.MethodPost, url, r)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	return res
}

func getJSON(url, token string) map[string]any {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		log.Fatalf("GET %s -> %d %s", url, res.StatusCode, raw)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		log.Fatalf("decode %s: %v (%s)", url, err, raw)
	}
	return out
}

func mustStatus(want int, res *http.Response) {
	defer res.Body.Close()
	if res.StatusCode != want {
		raw, _ := io.ReadAll(res.Body)
		log.Fatalf("status %d want %d %s", res.StatusCode, want, raw)
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
