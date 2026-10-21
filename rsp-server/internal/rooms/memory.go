package rooms

import (
	"context"
	"sync"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/google/uuid"
)

// Memory is an in-process Store. Used by tests and as the uniqueness source
// the hub already holds; production wraps Postgres.
type Memory struct {
	mu      sync.Mutex
	users   map[string]ident.Player
	rooms   map[string]*memRoom
	byCode  map[string]string // passcode → id, open rooms only
	matches map[string]*MemMatch
	stats   map[string]MemStats
	now     func() time.Time
}

// MemMatch is in-memory persistence for tests (and hub play without Postgres).
type MemMatch struct {
	ID       string
	RoomID   string
	WinnerID string
	Ended    bool
	Rounds   []MemRound
}

type MemRound struct {
	Number   int
	WinnerID string
	Moves    map[string]string
}

type MemStats struct {
	Played, Wins, Losses  int
	Rock, Paper, Scissors int
}

type memRoom struct {
	Room
	playerIDs []string
}

func NewMemory() *Memory {
	return &Memory{
		users:   make(map[string]ident.Player),
		rooms:   make(map[string]*memRoom),
		byCode:  make(map[string]string),
		matches: make(map[string]*MemMatch),
		stats:   make(map[string]MemStats),
		now:     time.Now,
	}
}

func (m *Memory) EnsureGuest(_ context.Context, p ident.Player) (ident.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if existing, ok := m.users[p.ID]; ok {
		if p.DisplayName != "" && p.DisplayName != ident.DefaultName {
			existing.DisplayName = p.DisplayName
		}
		if p.Avatar != "" {
			existing.Avatar = p.Avatar
		}
		if p.Color != "" {
			existing.Color = p.Color
		}
		m.users[p.ID] = existing
		return existing, nil
	}
	if p.DisplayName == "" {
		p.DisplayName = ident.DefaultName
	}
	if p.Avatar == "" {
		p.Avatar = ident.DefaultAvatar
	}
	if p.Color == "" {
		p.Color = ident.DefaultColor
	}
	p.IsGuest = true
	m.users[p.ID] = p
	return p, nil
}

func (m *Memory) InsertRoom(_ context.Context, hostID, passcode string, wins, seconds int) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, taken := m.byCode[passcode]; taken {
		return Room{}, ErrPasscodeTaken
	}
	id := uuid.NewString()
	r := &memRoom{
		Room: Room{
			ID:           id,
			Passcode:     passcode,
			HostID:       hostID,
			WinsToFinish: wins,
			TurnSeconds:  seconds,
			Status:       StatusWaiting,
			CreatedAt:    m.now(),
		},
	}
	m.rooms[id] = r
	m.byCode[passcode] = id
	return r.Room, nil
}

func (m *Memory) GetRoom(_ context.Context, id string) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[id]
	if !ok {
		return Room{}, ErrNotFound
	}
	return m.snapshotLocked(r), nil
}

func (m *Memory) GetOpenByPasscode(_ context.Context, passcode string) (Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byCode[passcode]
	if !ok {
		return Room{}, ErrWrongPasscode
	}
	r := m.rooms[id]
	if r == nil || r.Status == StatusClosed {
		return Room{}, ErrWrongPasscode
	}
	return m.snapshotLocked(r), nil
}

func (m *Memory) AddPlayer(_ context.Context, roomID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	for _, id := range r.playerIDs {
		if id == userID {
			return nil
		}
	}
	r.playerIDs = append(r.playerIDs, userID)
	return nil
}

func (m *Memory) RemovePlayer(_ context.Context, roomID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	out := r.playerIDs[:0]
	for _, id := range r.playerIDs {
		if id != userID {
			out = append(out, id)
		}
	}
	r.playerIDs = out
	return nil
}

func (m *Memory) ListPlayers(_ context.Context, roomID string) ([]Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, ErrNotFound
	}
	return m.playersLocked(r), nil
}

func (m *Memory) CloseRoom(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[id]
	if !ok {
		return ErrNotFound
	}
	r.Status = StatusClosed
	delete(m.byCode, r.Passcode)
	return nil
}

func (m *Memory) SetHost(_ context.Context, roomID, hostID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return ErrNotFound
	}
	r.HostID = hostID
	return nil
}

func (m *Memory) SetStatus(_ context.Context, id string, status Status) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[id]
	if !ok {
		return ErrNotFound
	}
	r.Status = status
	if status == StatusClosed {
		delete(m.byCode, r.Passcode)
	}
	return nil
}

func (m *Memory) InsertMatch(_ context.Context, roomID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rooms[roomID]; !ok {
		return "", ErrNotFound
	}
	id := uuid.NewString()
	m.matches[id] = &MemMatch{ID: id, RoomID: roomID}
	return id, nil
}

func (m *Memory) FinishMatch(_ context.Context, matchID, winnerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	mm, ok := m.matches[matchID]
	if !ok {
		return ErrNotFound
	}
	mm.WinnerID = winnerID
	mm.Ended = true
	return nil
}

func (m *Memory) SaveRound(_ context.Context, matchID string, roundNo int, winnerID string, moves map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	mm, ok := m.matches[matchID]
	if !ok {
		return ErrNotFound
	}
	copied := make(map[string]string, len(moves))
	for k, v := range moves {
		copied[k] = v
	}
	mm.Rounds = append(mm.Rounds, MemRound{Number: roundNo, WinnerID: winnerID, Moves: copied})
	return nil
}

func (m *Memory) ApplyMatchStats(_ context.Context, userID string, won, lost bool, rock, paper, scissors int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.stats[userID]
	s.Played++
	if won {
		s.Wins++
	}
	if lost {
		s.Losses++
	}
	s.Rock += rock
	s.Paper += paper
	s.Scissors += scissors
	m.stats[userID] = s
	return nil
}

func (m *Memory) Match(id string) (MemMatch, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mm, ok := m.matches[id]
	if !ok {
		return MemMatch{}, false
	}
	out := *mm
	out.Rounds = append([]MemRound(nil), mm.Rounds...)
	return out, true
}

func (m *Memory) Stats(userID string) MemStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats[userID]
}

func (m *Memory) snapshotLocked(r *memRoom) Room {
	out := r.Room
	out.Players = m.playersLocked(r)
	return out
}

func (m *Memory) playersLocked(r *memRoom) []Player {
	out := make([]Player, 0, len(r.playerIDs))
	for _, id := range r.playerIDs {
		p := m.users[id]
		out = append(out, Player{
			ID:          id,
			DisplayName: p.DisplayName,
			Avatar:      p.Avatar,
			Color:       p.Color,
			IsHost:      id == r.HostID,
			Joined:      true,
		})
	}
	return out
}
