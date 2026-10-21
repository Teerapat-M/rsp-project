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
	fmt.Println("smoke ok: first-to-1 match, timeout auto-pick, stats, and history against", base)
}

func playSubmittedMatch(base, wsBase string) {
	a, _, roomID, wa, wb := seatAndConnect(base, wsBase, 1, 15)
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
	a, _, roomID, wa, wb := seatAndConnect(base, wsBase, 1, 1)
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

func seatAndConnect(base, wsBase string, winsToFinish, turnSeconds int) (session, session, string, *websocket.Conn, *websocket.Conn) {
	a := guest(base, "SmokeA")
	b := guest(base, "SmokeB")
	room := postJSON(base+"/rooms", a.Access, map[string]int{"winsToFinish": winsToFinish, "turnSeconds": turnSeconds})
	_ = postJSON(base+"/rooms/join", b.Access, map[string]string{"passcode": str(room["passcode"])})
	roomID := str(room["id"])

	wa := dialWS(wsBase + "/ws?token=" + a.Access + "&room=" + roomID)
	wb := dialWS(wsBase + "/ws?token=" + b.Access + "&room=" + roomID)
	// Attach is complete once the connect snapshot arrives. Starting before
	// that races the hub broadcast of match.started.
	waitType(wa, "room.updated")
	waitType(wb, "room.updated")
	return a, b, roomID, wa, wb
}

type session struct {
	Access string
	ID     string
}

func guest(base, name string) session {
	raw := postJSON(base+"/auth/guest", "", map[string]string{"displayName": name, "avatar": "rock", "color": "#9DB89A"})
	user, _ := raw["user"].(map[string]any)
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
