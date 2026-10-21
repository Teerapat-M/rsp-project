package hub

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 54 * time.Second
	maxMsg     = 4096
)

// ErrPhase3 is returned when play/start handlers are unset.
var ErrPhase3 = errors.New("not implemented")

func (h *Hub) upgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}
}

// ClientMessage is the JSON body the browser sends.
type ClientMessage struct {
	Type string `json:"type"`
	Move string `json:"move,omitempty"`
}

type client struct {
	hub    *Hub
	sess   *session
	player ident.Player
	ws     *websocket.Conn
	send   chan []byte

	closeOnce sync.Once
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.send)
	})
}

// ServeWS upgrades the connection. Caller already authenticated the player.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, roomID string, player ident.Player) {
	ctx := r.Context()
	s := h.get(roomID)
	if s == nil {
		var err error
		s, err = h.load(ctx, roomID)
		if err != nil {
			http.Error(w, `{"code":"NOT_FOUND","message":"Room not found"}`, http.StatusNotFound)
			return
		}
	}

	s.mu.Lock()
	_, inRoom := s.findLocked(player.ID)
	closed := s.status == rooms.StatusClosed
	s.mu.Unlock()
	if closed || !inRoom {
		http.Error(w, `{"code":"NOT_FOUND","message":"Not in this room"}`, http.StatusForbidden)
		return
	}

	up := h.upgrader()
	up.Subprotocols = websocket.Subprotocols(r)
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := h.attach(s, player, conn)
	if c == nil {
		_ = conn.Close()
		return
	}
	go c.writePump()
	c.readPump()
}

func (c *client) readPump() {
	defer func() {
		c.hub.detach(c)
		_ = c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMsg)
	_ = c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		return c.ws.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		c.hub.handleClient(c, data)
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.ws.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) handleClient(c *client, data []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendErr("VALIDATION", "invalid JSON")
		return
	}
	switch msg.Type {
	case "ping":
		payload, _ := json.Marshal(event{"type": "pong"})
		select {
		case c.send <- payload:
		default:
		}
	case "leave":
		_, _ = h.Leave(context.Background(), c.sess.id, c.player.ID)
	case "lobby.ready", "ready":
		st := c.sess.statusSnapshot()
		if st == rooms.StatusWaiting {
			_, _ = h.SetLobbyReady(c.sess.id, c.player.ID, true)
			return
		}
		h.playOrReject(c, msg)
	case "lobby.unready":
		_, _ = h.SetLobbyReady(c.sess.id, c.player.ID, false)
	case "submit", "claim_win", "play_again":
		h.playOrReject(c, msg)
	default:
		c.sendErr("VALIDATION", "unknown message type")
	}
}

func (h *Hub) playOrReject(c *client, msg ClientMessage) {
	if h.onPlay == nil {
		c.sendErr("NOT_IMPLEMENTED", ErrPhase3.Error())
		return
	}
	h.onPlay(c.sess.id, c.player.ID, msg)
}

func (c *client) sendErr(code, message string) {
	payload, _ := json.Marshal(event{"type": "error", "code": code, "message": message})
	select {
	case c.send <- payload:
	default:
	}
}

func (s *session) statusSnapshot() rooms.Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}
