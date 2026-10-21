// Package rooms is create/join/leave rules plus persistence.
//
// Live presence (sockets, ready flags, disconnect grace) lives in internal/hub.
// This package is the bit you can test without a WebSocket: passcode uniqueness,
// two-player cap, wrong passcode, idempotent re-join.
package rooms

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/ident"
)

const MaxPlayers = 2

var (
	ErrWrongPasscode = errors.New("wrong passcode")
	ErrRoomFull      = errors.New("room is full")
	ErrNotFound      = errors.New("room not found")
	ErrNotInRoom     = errors.New("not in this room")
	ErrPasscodeTaken = errors.New("passcode taken")
	ErrInvalidWins   = errors.New("wins_to_finish must be >= 1")
	ErrInvalidTurn   = errors.New("turn_seconds must be >= 1")
	ErrClosed        = errors.New("room is closed")
)

type Status string

const (
	StatusWaiting Status = "waiting"
	StatusPlaying Status = "playing"
	StatusClosed  Status = "closed"
)

// Player is a room member as the frontend RoomPlayer type expects.
type Player struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
	Color       string `json:"color"`
	IsHost      bool   `json:"isHost"`
	Joined      bool   `json:"joined"`
	Ready       bool   `json:"ready"`
	Connected   bool   `json:"connected"`
}

// Room matches rsp-client/src/types.ts Room (camelCase) plus presence extras.
type Room struct {
	ID           string    `json:"id"`
	Passcode     string    `json:"passcode"`
	HostID       string    `json:"hostId"`
	WinsToFinish int       `json:"winsToFinish"`
	TurnSeconds  int       `json:"turnSeconds"`
	Status       Status    `json:"status"`
	Players      []Player  `json:"players"`
	CreatedAt    time.Time `json:"-"`
}

func ValidateSettings(winsToFinish, turnSeconds int) error {
	if winsToFinish < 1 {
		return ErrInvalidWins
	}
	if turnSeconds < 1 {
		return ErrInvalidTurn
	}
	return nil
}

func NormalizePasscode(code string) string {
	var b strings.Builder
	for _, r := range code {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// CheckJoin is the two-player rule. Already-in is a no-op (reconnect / double POST).
func CheckJoin(memberIDs []string, candidate string) error {
	for _, id := range memberIDs {
		if id == candidate {
			return nil
		}
	}
	if len(memberIDs) >= MaxPlayers {
		return ErrRoomFull
	}
	return nil
}

func MemberIDs(players []Player) []string {
	out := make([]string, len(players))
	for i, p := range players {
		out[i] = p.ID
	}
	return out
}

func FromIdent(p ident.Player, isHost, ready, connected bool) Player {
	return Player{
		ID:          p.ID,
		DisplayName: p.DisplayName,
		Avatar:      p.Avatar,
		Color:       p.Color,
		IsHost:      isHost,
		Joined:      true,
		Ready:       ready,
		Connected:   connected,
	}
}

// Store persists rooms. The hub holds live sockets; the store is Postgres in
// production and Memory in tests.
type Store interface {
	EnsureGuest(ctx context.Context, p ident.Player) (ident.Player, error)
	InsertRoom(ctx context.Context, hostID, passcode string, wins, seconds int) (Room, error)
	GetRoom(ctx context.Context, id string) (Room, error)
	GetOpenByPasscode(ctx context.Context, passcode string) (Room, error)
	AddPlayer(ctx context.Context, roomID, userID string) error
	RemovePlayer(ctx context.Context, roomID, userID string) error
	ListPlayers(ctx context.Context, roomID string) ([]Player, error)
	CloseRoom(ctx context.Context, id string) error
	SetHost(ctx context.Context, roomID, hostID string) error
	SetStatus(ctx context.Context, id string, status Status) error
}
