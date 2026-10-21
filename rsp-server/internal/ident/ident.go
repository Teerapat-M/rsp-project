// Package ident is a TEMPORARY player identity until phase 4 JWT auth.
//
// Resolution order:
//  1. X-Player-ID header
//  2. Authorization: Bearer <token>
//  3. ?token= query (WebSocket)
//
// A UUID is used as-is. Any other string is hashed to a stable UUID (v5) so
// curl -H 'X-Player-ID: alice' stays the same player across requests.
// The Bearer token for a minted guest is that UUID (a "dev guest token").
package ident

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const (
	HeaderPlayerID     = "X-Player-ID"
	HeaderPlayerName   = "X-Player-Name"
	HeaderPlayerAvatar = "X-Player-Avatar"
	HeaderPlayerColor  = "X-Player-Color"

	DefaultAvatar = "rock"
	DefaultColor  = "#9DB89A"
	DefaultName   = "Guest"
)

// DevNamespace is the v5 namespace for non-UUID player keys. Stable forever.
var DevNamespace = uuid.MustParse("a51a0c2e-7b4d-4f8e-9c1a-2d3e4f506070")

var ErrNoPlayer = errors.New("missing player identity")

// Player is who the server treats as the caller. Avatar/color are not in Postgres
// yet (phase 4/5); they live on the in-memory hub so the frontend JSON matches.
type Player struct {
	ID          string
	DisplayName string
	Avatar      string
	Color       string
	IsGuest     bool
}

func (p Player) withDefaults() Player {
	if p.DisplayName == "" {
		p.DisplayName = DefaultName
	}
	if p.Avatar == "" {
		p.Avatar = DefaultAvatar
	}
	if p.Color == "" {
		p.Color = DefaultColor
	}
	p.IsGuest = true
	return p
}

// Normalize turns a header/token into a users.id UUID.
func Normalize(raw string) uuid.UUID {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "dev:")
	if u, err := uuid.Parse(raw); err == nil {
		return u
	}
	return uuid.NewSHA1(DevNamespace, []byte(raw))
}

// FromRequest reads the temporary identity. It does not hit the database.
func FromRequest(r *http.Request) (Player, error) {
	raw := strings.TrimSpace(r.Header.Get(HeaderPlayerID))
	if raw == "" {
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			raw = strings.TrimSpace(auth[7:])
		}
	}
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if raw == "" {
		return Player{}, ErrNoPlayer
	}

	name := strings.TrimSpace(r.Header.Get(HeaderPlayerName))
	if name == "" {
		if _, err := uuid.Parse(strings.TrimPrefix(strings.TrimSpace(raw), "dev:")); err != nil {
			name = strings.TrimSpace(raw)
		}
	}

	return Player{
		ID:          Normalize(raw).String(),
		DisplayName: name,
		Avatar:      strings.TrimSpace(r.Header.Get(HeaderPlayerAvatar)),
		Color:       strings.TrimSpace(r.Header.Get(HeaderPlayerColor)),
	}.withDefaults(), nil
}
