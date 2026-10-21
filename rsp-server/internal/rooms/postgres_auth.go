package rooms

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/sqlcdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (p *Postgres) CreateGuest(ctx context.Context, displayName, avatar, color string) (ident.Player, error) {
	if avatar == "" {
		avatar = ident.DefaultAvatar
	}
	if color == "" {
		color = ident.DefaultColor
	}
	id := uuid.New()
	row, err := p.q.InsertGuest(ctx, sqlcdb.InsertGuestParams{
		ID:          id,
		DisplayName: displayName,
		Avatar:      avatar,
		Color:       color,
	})
	if err != nil {
		return ident.Player{}, err
	}
	return PlayerFromUser(row), nil
}

func (p *Postgres) CreatePasswordUser(ctx context.Context, username, hash, email, displayName, avatar, color string) (ident.Player, error) {
	if avatar == "" {
		avatar = ident.DefaultAvatar
	}
	if color == "" {
		color = ident.DefaultColor
	}
	var em *string
	if email != "" {
		em = &email
	}
	row, err := p.q.InsertUser(ctx, sqlcdb.InsertUserParams{
		DisplayName:  displayName,
		Email:        em,
		IsGuest:      false,
		Username:     &username,
		PasswordHash: &hash,
		Avatar:       avatar,
		Color:        color,
	})
	if err != nil {
		return ident.Player{}, mapAuthUnique(err)
	}
	return PlayerFromUser(row), nil
}

func (p *Postgres) GetPlayer(ctx context.Context, id string) (ident.Player, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return ident.Player{}, ErrNotFound
	}
	row, err := p.q.GetUser(ctx, uid)
	if err != nil {
		return ident.Player{}, mapNotFound(err)
	}
	return PlayerFromUser(row), nil
}

func (p *Postgres) GetPlayerByUsername(ctx context.Context, username string) (ident.Player, string, error) {
	row, err := p.q.GetUserByUsername(ctx, username)
	if err != nil {
		return ident.Player{}, "", mapNotFound(err)
	}
	return PlayerFromUser(row), deref(row.PasswordHash), nil
}

func (p *Postgres) GetPlayerByEmail(ctx context.Context, email string) (ident.Player, string, error) {
	row, err := p.q.GetUserByEmail(ctx, email)
	if err != nil {
		return ident.Player{}, "", mapNotFound(err)
	}
	return PlayerFromUser(row), deref(row.PasswordHash), nil
}

func (p *Postgres) UpgradeGuest(ctx context.Context, id, username, hash, email, displayName string) (ident.Player, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return ident.Player{}, ErrNotFound
	}
	var em *string
	if email != "" {
		em = &email
	}
	row, err := p.q.UpgradeGuest(ctx, sqlcdb.UpgradeGuestParams{
		ID:           uid,
		Username:     username,
		PasswordHash: hash,
		Email:        em,
		DisplayName:  displayName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ident.Player{}, ErrNotGuest
		}
		return ident.Player{}, mapAuthUnique(err)
	}
	return PlayerFromUser(row), nil
}

func (p *Postgres) SetPassword(ctx context.Context, id, hash string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return ErrNotFound
	}
	return p.q.SetPasswordHash(ctx, sqlcdb.SetPasswordHashParams{ID: uid, PasswordHash: hash})
}

func (p *Postgres) LinkProvider(ctx context.Context, userID, provider, providerUserID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrNotFound
	}
	_, err = p.q.InsertAuthProvider(ctx, sqlcdb.InsertAuthProviderParams{
		UserID:         uid,
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	return err
}

func (p *Postgres) GetByProvider(ctx context.Context, provider, providerUserID string) (ident.Player, error) {
	row, err := p.q.GetAuthProvider(ctx, sqlcdb.GetAuthProviderParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		return ident.Player{}, mapNotFound(err)
	}
	return p.GetPlayer(ctx, row.UserID.String())
}

func (p *Postgres) CreateGoogleUser(ctx context.Context, email, displayName, providerUserID, avatar, color string) (ident.Player, error) {
	if avatar == "" {
		avatar = ident.DefaultAvatar
	}
	if color == "" {
		color = ident.DefaultColor
	}
	var em *string
	if email != "" {
		em = &email
	}
	row, err := p.q.InsertUser(ctx, sqlcdb.InsertUserParams{
		DisplayName: displayName,
		Email:       em,
		IsGuest:     false,
		Avatar:      avatar,
		Color:       color,
	})
	if err != nil {
		return ident.Player{}, mapAuthUnique(err)
	}
	if _, err := p.q.InsertAuthProvider(ctx, sqlcdb.InsertAuthProviderParams{
		UserID:         row.ID,
		Provider:       "google",
		ProviderUserID: providerUserID,
	}); err != nil {
		return ident.Player{}, err
	}
	return PlayerFromUser(row), nil
}

func (p *Postgres) InsertRefresh(ctx context.Context, userID, hash string, expires time.Time) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrNotFound
	}
	_, err = p.q.InsertRefreshToken(ctx, sqlcdb.InsertRefreshTokenParams{
		UserID:    uid,
		TokenHash: hash,
		ExpiresAt: expires,
	})
	return err
}

func (p *Postgres) GetRefresh(ctx context.Context, hash string) (userID string, expires, created time.Time, revoked bool, err error) {
	row, err := p.q.GetRefreshToken(ctx, hash)
	if err != nil {
		return "", time.Time{}, time.Time{}, false, mapNotFound(err)
	}
	revoked = row.RevokedAt.Valid
	return row.UserID.String(), row.ExpiresAt.Time, row.CreatedAt.Time, revoked, nil
}

func (p *Postgres) RevokeRefresh(ctx context.Context, hash string) error {
	return p.q.RevokeRefreshToken(ctx, hash)
}

func (p *Postgres) RevokeAllRefresh(ctx context.Context, userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrNotFound
	}
	return p.q.RevokeUserRefreshTokens(ctx, uid)
}

func (p *Postgres) InsertReset(ctx context.Context, userID, hash string, expires time.Time) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return ErrNotFound
	}
	_, err = p.q.InsertPasswordReset(ctx, sqlcdb.InsertPasswordResetParams{
		UserID:    uid,
		TokenHash: hash,
		ExpiresAt: expires,
	})
	return err
}

func (p *Postgres) ConsumeReset(ctx context.Context, hash string) (string, error) {
	row, err := p.q.GetPasswordReset(ctx, hash)
	if err != nil {
		return "", mapNotFound(err)
	}
	if row.UsedAt.Valid || !row.ExpiresAt.Valid || time.Now().After(row.ExpiresAt.Time) {
		return "", ErrNotFound
	}
	if err := p.q.MarkPasswordResetUsed(ctx, row.ID); err != nil {
		return "", err
	}
	return row.UserID.String(), nil
}

func mapAuthUnique(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	switch pgErr.ConstraintName {
	case "users_email_key":
		return ErrEmailTaken
	case "users_username_lower_idx":
		return ErrUsernameTaken
	default:
		blob := strings.ToLower(pgErr.Detail + " " + pgErr.Message)
		if strings.Contains(blob, "email") {
			return ErrEmailTaken
		}
		return ErrUsernameTaken
	}
}
