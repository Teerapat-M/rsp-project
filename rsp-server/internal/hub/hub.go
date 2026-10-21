// Package hub is the in-memory live-room manager.
//
// One mutex on the map, one mutex per room. Fine at ~32 pairs.
// Empty rooms are closed (passcode reusable) and dropped from the maps.
package hub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/game"
	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/passcode"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"github.com/gorilla/websocket"
)

const (
	defaultGrace       = 20 * time.Second
	defaultResultPause = 3 * time.Second
)

// PlayHandler is the phase 3 hook for submit / round-ready / claim_win / play_again.
// Phase 2 leaves this nil: those client messages get a typed error, not a crash.
type PlayHandler func(roomID, playerID string, msg ClientMessage)

// StartHandler is the phase 3 hook for POST /rooms/:id/start.
type StartHandler func(roomID, playerID string) error

type Config struct {
	Grace time.Duration
	// After schedules f after d and returns a stop func (time.AfterFunc in prod).
	After func(d time.Duration, f func()) (stop func() bool)
	// ResultPause is the delay after round.resolved before the next clock
	// (or match.finished). Zero means 3s. Tests can set a tiny duration.
	ResultPause time.Duration
	OnPlay      PlayHandler
	OnStart     StartHandler
	// Pick replaces crypto/rand when a turn times out with no card. Tests
	// inject a stub so auto-picks are deterministic.
	Pick game.Pick
}

type Hub struct {
	store       rooms.Store
	grace       time.Duration
	after       func(d time.Duration, f func()) (stop func() bool)
	resultPause time.Duration
	onPlay      PlayHandler
	onStart     StartHandler
	pick        game.Pick

	mu     sync.Mutex
	rooms  map[string]*session
	byCode map[string]*session
}

func New(store rooms.Store, cfg Config) *Hub {
	grace := cfg.Grace
	if grace <= 0 {
		grace = defaultGrace
	}
	after := cfg.After
	if after == nil {
		after = func(d time.Duration, f func()) func() bool {
			t := time.AfterFunc(d, f)
			return t.Stop
		}
	}
	pause := cfg.ResultPause
	if pause == 0 {
		pause = defaultResultPause
	}
	h := &Hub{
		store:       store,
		grace:       grace,
		after:       after,
		resultPause: pause,
		onPlay:      cfg.OnPlay,
		onStart:     cfg.OnStart,
		pick:        cfg.Pick,
		rooms:       make(map[string]*session),
		byCode:      make(map[string]*session),
	}
	if h.onPlay == nil {
		h.onPlay = h.handlePlay
	}
	if h.onStart == nil {
		h.onStart = func(roomID, playerID string) error {
			return h.beginMatch(roomID, playerID, false)
		}
	}
	return h
}

func (h *Hub) Store() rooms.Store   { return h.store }
func (h *Hub) Grace() time.Duration { return h.grace }

func (h *Hub) StartMatch(roomID, playerID string) error {
	if h.onStart == nil {
		return ErrPhase3
	}
	return h.onStart(roomID, playerID)
}

type member struct {
	player    ident.Player
	ready     bool
	conns     map[*client]struct{}
	graceStop func() bool
}

type session struct {
	mu       sync.Mutex
	id       string
	passcode string
	hostID   string
	wins     int
	seconds  int
	status   rooms.Status
	members  []*member // join order

	match       *game.Match
	matchID     string
	remaining   int
	paused      bool
	timeoutStop func() bool
	tickStop    func() bool
	resultStop  func() bool
	playAgain   map[string]bool
	clockRound  int
}

func (h *Hub) Create(ctx context.Context, host ident.Player, wins, seconds int) (rooms.Room, error) {
	if err := rooms.ValidateSettings(wins, seconds); err != nil {
		return rooms.Room{}, err
	}
	host, err := h.store.EnsureGuest(ctx, host)
	if err != nil {
		return rooms.Room{}, err
	}

	var persisted rooms.Room
	for i := 0; i < 12; i++ {
		code := passcode.New()
		persisted, err = h.store.InsertRoom(ctx, host.ID, code, wins, seconds)
		if err == nil {
			break
		}
		if errors.Is(err, rooms.ErrPasscodeTaken) {
			continue
		}
		return rooms.Room{}, err
	}
	if err != nil {
		return rooms.Room{}, passcode.ErrExhausted
	}
	if err := h.store.AddPlayer(ctx, persisted.ID, host.ID); err != nil {
		return rooms.Room{}, err
	}

	s := &session{
		id:       persisted.ID,
		passcode: persisted.Passcode,
		hostID:   host.ID,
		wins:     wins,
		seconds:  seconds,
		status:   rooms.StatusWaiting,
		members:  []*member{{player: host, conns: make(map[*client]struct{})}},
	}
	h.mu.Lock()
	h.rooms[s.id] = s
	h.byCode[s.passcode] = s
	h.mu.Unlock()
	return s.snapshot(), nil
}

func (h *Hub) Join(ctx context.Context, player ident.Player, rawCode string) (rooms.Room, error) {
	code := rooms.NormalizePasscode(rawCode)
	if !passcode.Valid(code) {
		return rooms.Room{}, rooms.ErrWrongPasscode
	}
	player, err := h.store.EnsureGuest(ctx, player)
	if err != nil {
		return rooms.Room{}, err
	}

	s, err := h.sessionByPasscode(ctx, code)
	if err != nil {
		return rooms.Room{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// ponytail: room mutex is held across the store write. Fine at ~32 pairs.
	// Upgrade: persist outside the lock if Postgres ever becomes the slow path.
	if s.status == rooms.StatusClosed {
		return rooms.Room{}, rooms.ErrWrongPasscode
	}
	ids := s.memberIDs()
	if err := rooms.CheckJoin(ids, player.ID); err != nil {
		return rooms.Room{}, err
	}
	if !containsID(ids, player.ID) {
		if err := h.store.AddPlayer(ctx, s.id, player.ID); err != nil {
			return rooms.Room{}, err
		}
		s.members = append(s.members, &member{player: player, conns: make(map[*client]struct{})})
		snap := s.snapshotLocked()
		s.broadcastLocked(event{"type": "player.joined", "player": snap.Players[len(snap.Players)-1]})
		s.broadcastLocked(event{"type": "room.updated", "room": snap})
		return snap, nil
	}
	s.touchPlayerLocked(player)
	return s.snapshotLocked(), nil
}

func (h *Hub) Leave(ctx context.Context, roomID, playerID string) (rooms.Room, error) {
	s := h.get(roomID)
	if s == nil {
		loaded, err := h.load(ctx, roomID)
		if err != nil {
			return rooms.Room{}, err
		}
		s = loaded
	}

	s.mu.Lock()
	if _, ok := s.findLocked(playerID); !ok {
		snap := s.snapshotLocked()
		s.mu.Unlock()
		return snap, nil
	}
	if s.match != nil && s.match.Status() == game.StatusPlaying {
		h.forfeitLocked(s, playerID)
	}
	s.removeLocked(playerID)
	if err := h.store.RemovePlayer(ctx, s.id, playerID); err != nil {
		log.Printf("hub: remove player: %v", err)
	}
	empty := len(s.members) == 0
	if !empty && s.hostID == playerID {
		s.hostID = s.members[0].player.ID
		if err := h.store.SetHost(ctx, s.id, s.hostID); err != nil {
			log.Printf("hub: set host: %v", err)
		}
	}
	if empty {
		s.status = rooms.StatusClosed
		if err := h.store.CloseRoom(ctx, s.id); err != nil {
			log.Printf("hub: close room: %v", err)
		}
	}
	snap := s.snapshotLocked()
	s.broadcastLocked(event{"type": "player.left", "playerId": playerID})
	s.broadcastLocked(event{"type": "room.updated", "room": snap})
	if empty {
		s.closeAllLocked()
	}
	s.mu.Unlock()

	if empty {
		h.drop(s)
	}
	return snap, nil
}

func (h *Hub) Get(ctx context.Context, roomID string) (rooms.Room, error) {
	s := h.get(roomID)
	if s == nil {
		var err error
		s, err = h.load(ctx, roomID)
		if err != nil {
			return rooms.Room{}, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked(), nil
}

func (h *Hub) SetLobbyReady(roomID, playerID string, ready bool) (rooms.Room, error) {
	s := h.get(roomID)
	if s == nil {
		return rooms.Room{}, rooms.ErrNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.findLocked(playerID)
	if !ok {
		return rooms.Room{}, rooms.ErrNotInRoom
	}
	m.ready = ready
	snap := s.snapshotLocked()
	n := 0
	for _, mem := range s.members {
		if mem.ready {
			n++
		}
	}
	s.broadcastLocked(event{
		"type":        "lobby.ready",
		"playerId":    playerID,
		"ready":       n,
		"need":        rooms.MaxPlayers,
		"playerReady": ready,
	})
	s.broadcastLocked(event{"type": "room.updated", "room": snap})
	return snap, nil
}

func (h *Hub) LiveCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.rooms)
}

func (h *Hub) get(id string) *session {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.rooms[id]
}

func (h *Hub) sessionByPasscode(ctx context.Context, code string) (*session, error) {
	h.mu.Lock()
	if s := h.byCode[code]; s != nil {
		h.mu.Unlock()
		return s, nil
	}
	h.mu.Unlock()

	persisted, err := h.store.GetOpenByPasscode(ctx, code)
	if err != nil {
		return nil, err
	}
	return h.adopt(ctx, persisted)
}

func (h *Hub) load(ctx context.Context, id string) (*session, error) {
	persisted, err := h.store.GetRoom(ctx, id)
	if err != nil {
		return nil, err
	}
	return h.adopt(ctx, persisted)
}

func (h *Hub) adopt(ctx context.Context, persisted rooms.Room) (*session, error) {
	players, err := h.store.ListPlayers(ctx, persisted.ID)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if s := h.rooms[persisted.ID]; s != nil {
		return s, nil
	}
	if persisted.Status == rooms.StatusClosed {
		return nil, rooms.ErrNotFound
	}
	s := &session{
		id:       persisted.ID,
		passcode: persisted.Passcode,
		hostID:   persisted.HostID,
		wins:     persisted.WinsToFinish,
		seconds:  persisted.TurnSeconds,
		status:   persisted.Status,
	}
	for _, p := range players {
		s.members = append(s.members, &member{
			player: ident.Player{
				ID:          p.ID,
				DisplayName: p.DisplayName,
				Avatar:      orDefault(p.Avatar, ident.DefaultAvatar),
				Color:       orDefault(p.Color, ident.DefaultColor),
			},
			conns: make(map[*client]struct{}),
		})
	}
	h.rooms[s.id] = s
	if s.status != rooms.StatusClosed {
		h.byCode[s.passcode] = s
	}
	return s, nil
}

func (h *Hub) drop(s *session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.members) > 0 {
		return
	}
	delete(h.rooms, s.id)
	delete(h.byCode, s.passcode)
}

func (s *session) memberIDs() []string {
	out := make([]string, len(s.members))
	for i, m := range s.members {
		out[i] = m.player.ID
	}
	return out
}

func (s *session) findLocked(id string) (*member, bool) {
	for _, m := range s.members {
		if m.player.ID == id {
			return m, true
		}
	}
	return nil, false
}

func (s *session) touchPlayerLocked(p ident.Player) {
	m, ok := s.findLocked(p.ID)
	if !ok {
		return
	}
	if p.DisplayName != "" {
		m.player.DisplayName = p.DisplayName
	}
	if p.Avatar != "" {
		m.player.Avatar = p.Avatar
	}
	if p.Color != "" {
		m.player.Color = p.Color
	}
}

func (s *session) removeLocked(playerID string) {
	out := s.members[:0]
	for _, m := range s.members {
		if m.player.ID != playerID {
			out = append(out, m)
			continue
		}
		if m.graceStop != nil {
			m.graceStop()
			m.graceStop = nil
		}
		for c := range m.conns {
			c.close()
		}
	}
	s.members = out
}

func (s *session) closeAllLocked() {
	for _, m := range s.members {
		for c := range m.conns {
			c.close()
		}
	}
}

func (s *session) snapshot() rooms.Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *session) sendToLocked(playerID string, msg event) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	m, ok := s.findLocked(playerID)
	if !ok {
		return
	}
	for c := range m.conns {
		select {
		case c.send <- payload:
		default:
			c.close()
		}
	}
}

func (s *session) snapshotLocked() rooms.Room {
	ps := make([]rooms.Player, 0, len(s.members))
	for _, m := range s.members {
		ps = append(ps, rooms.FromIdent(m.player, m.player.ID == s.hostID, m.ready, len(m.conns) > 0))
	}
	return rooms.Room{
		ID:           s.id,
		Passcode:     s.passcode,
		HostID:       s.hostID,
		WinsToFinish: s.wins,
		TurnSeconds:  s.seconds,
		Status:       s.status,
		Players:      ps,
	}
}

type event map[string]any

func (s *session) broadcastLocked(msg event) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	for _, m := range s.members {
		for c := range m.conns {
			select {
			case c.send <- payload:
			default:
				c.close()
			}
		}
	}
}

func containsID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func (h *Hub) attach(s *session, player ident.Player, conn *websocket.Conn) *client {
	c := &client{
		hub:    h,
		sess:   s,
		player: player,
		ws:     conn,
		send:   make(chan []byte, 32),
	}
	s.mu.Lock()
	m, ok := s.findLocked(player.ID)
	if !ok {
		s.mu.Unlock()
		return nil
	}
	s.touchPlayerLocked(player)
	reconnected := m.graceStop != nil
	if m.graceStop != nil {
		m.graceStop()
		m.graceStop = nil
	}
	m.conns[c] = struct{}{}
	snap := s.snapshotLocked()
	s.broadcastLocked(event{
		"type":             "presence",
		"playerId":         player.ID,
		"connected":        true,
		"reconnectSeconds": nil,
	})
	if reconnected {
		s.broadcastLocked(event{"type": "player.reconnected", "playerId": player.ID})
		if s.match != nil && s.match.Status() == game.StatusPlaying {
			h.sendOpponentEventLocked(s, player.ID, "opponent.reconnected")
		}
	}
	s.broadcastLocked(event{"type": "room.updated", "room": snap})
	if s.match != nil {
		// Late and reconnecting sockets were not in the room at POST /start,
		// so they never saw match.started. Send the same snapshot now.
		typ := "view"
		switch s.match.Status() {
		case game.StatusPlaying:
			typ = "match.started"
		case game.StatusFinished:
			typ = "match.finished"
		}
		s.sendToLocked(player.ID, event{"type": typ, "view": h.playerViewLocked(s, player)})
		if reconnected && s.match.Status() == game.StatusPlaying && s.allConnectedLocked() {
			s.paused = false
			h.startClockLocked(s)
		}
	}
	s.mu.Unlock()
	return c
}

func (h *Hub) detach(c *client) {
	s := c.sess
	s.mu.Lock()
	m, ok := s.findLocked(c.player.ID)
	if !ok {
		s.mu.Unlock()
		return
	}
	delete(m.conns, c)
	c.close()
	if len(m.conns) > 0 {
		s.mu.Unlock()
		return
	}
	playerID := c.player.ID
	roomID := s.id
	grace := h.grace
	s.broadcastLocked(event{
		"type":             "presence",
		"playerId":         playerID,
		"connected":        false,
		"reconnectSeconds": int(grace / time.Second),
	})
	s.broadcastLocked(event{"type": "player.disconnected", "playerId": playerID, "reconnectSeconds": int(grace / time.Second)})
	s.broadcastLocked(event{"type": "room.updated", "room": s.snapshotLocked()})
	playing := s.match != nil && s.match.Status() == game.StatusPlaying
	if playing {
		s.paused = true
		s.stopClockLocked()
		h.sendOpponentEventLocked(s, playerID, "opponent.disconnected")
	}
	if m.graceStop != nil {
		m.graceStop()
	}
	m.graceStop = h.after(grace, func() {
		if playing {
			h.forfeit(roomID, playerID)
			return
		}
		if _, err := h.Leave(context.Background(), roomID, playerID); err != nil {
			log.Printf("hub: grace leave: %v", err)
		}
	})
	s.mu.Unlock()
}
