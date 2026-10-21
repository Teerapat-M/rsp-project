package auth

import (
	"context"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/ident"
)

// Users is the persistence the auth service needs. rooms.Memory and
// rooms.Postgres both implement it.
type Users interface {
	CreateGuest(ctx context.Context, displayName, avatar, color string) (ident.Player, error)
	CreatePasswordUser(ctx context.Context, username, hash, email, displayName, avatar, color string) (ident.Player, error)
	GetPlayer(ctx context.Context, id string) (ident.Player, error)
	GetPlayerByUsername(ctx context.Context, username string) (ident.Player, string, error)
	GetPlayerByEmail(ctx context.Context, email string) (ident.Player, string, error)
	UpgradeGuest(ctx context.Context, id, username, hash, email, displayName string) (ident.Player, error)
	SetPassword(ctx context.Context, id, hash string) error
	LinkProvider(ctx context.Context, userID, provider, providerUserID string) error
	GetByProvider(ctx context.Context, provider, providerUserID string) (ident.Player, error)
	CreateGoogleUser(ctx context.Context, email, displayName, providerUserID, avatar, color string) (ident.Player, error)
	InsertRefresh(ctx context.Context, userID, hash string, expires time.Time) error
	GetRefresh(ctx context.Context, hash string) (userID string, expires, created time.Time, revoked bool, err error)
	RevokeRefresh(ctx context.Context, hash string) error
	RevokeAllRefresh(ctx context.Context, userID string) error
	InsertReset(ctx context.Context, userID, hash string, expires time.Time) error
	ConsumeReset(ctx context.Context, hash string) (string, error)
}
