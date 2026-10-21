// Package ident is the player identity the hub and HTTP layer share.
//
// Production auth is a JWT (internal/auth). In APP_ENV=dev only, FromRequest
// still accepts X-Player-ID / a non-JWT bearer / ?token= so curl and older
// tests work. Those values are hashed to a stable UUID when they are not one.
package ident

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// LooksLikeJWT is three base64url segments separated by dots. Dev identity
// must not hash a failed access token into a new guest UUID.
func LooksLikeJWT(tok string) bool {
	n := 0
	for i := 0; i < len(tok); i++ {
		if tok[i] == '.' {
			n++
		}
	}
	return n == 2 && len(tok) > 8
}

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

// Player is who the server treats as the caller.
type Player struct {
	ID          string
	DisplayName string
	Username    string
	Email       string
	Avatar      string
	Color       string
	IsGuest     bool
}

func (p Player) WithDefaults() Player {
	if p.DisplayName == "" {
		p.DisplayName = DefaultName
	}
	if p.Avatar == "" {
		p.Avatar = DefaultAvatar
	}
	if p.Color == "" {
		p.Color = DefaultColor
	}
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
			tok := strings.TrimSpace(auth[7:])
			if !LooksLikeJWT(tok) {
				raw = tok
			}
		}
	}
	if raw == "" {
		tok := strings.TrimSpace(r.URL.Query().Get("token"))
		if !LooksLikeJWT(tok) {
			raw = tok
		}
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

	p := Player{
		ID:          Normalize(raw).String(),
		DisplayName: name,
		Avatar:      strings.TrimSpace(r.Header.Get(HeaderPlayerAvatar)),
		Color:       strings.TrimSpace(r.Header.Get(HeaderPlayerColor)),
		IsGuest:     true,
	}
	return p.WithDefaults(), nil
}
