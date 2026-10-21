package rooms

import (
	"context"
	"errors"
	"fmt"

	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/sqlcdb"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Postgres is the production Store. Unique passcode collisions surface as
// ErrPasscodeTaken so Hub.Create can retry.
type Postgres struct {
	q *sqlcdb.Queries
}

func NewPostgres(q *sqlcdb.Queries) *Postgres {
	return &Postgres{q: q}
}

func (p *Postgres) EnsureGuest(ctx context.Context, pl ident.Player) (ident.Player, error) {
	id, err := uuid.Parse(pl.ID)
	if err != nil {
		return ident.Player{}, fmt.Errorf("player id: %w", err)
	}
	name := pl.DisplayName
	if name == "" {
		name = ident.DefaultName
	}
	row, err := p.q.InsertGuest(ctx, sqlcdb.InsertGuestParams{ID: id, DisplayName: name})
	if err != nil {
		return ident.Player{}, err
	}
	pl.ID = row.ID.String()
	pl.DisplayName = row.DisplayName
	pl.IsGuest = row.IsGuest
	if pl.Avatar == "" {
		pl.Avatar = ident.DefaultAvatar
	}
	if pl.Color == "" {
		pl.Color = ident.DefaultColor
	}
	return pl, nil
}

func (p *Postgres) InsertRoom(ctx context.Context, hostID, passcode string, wins, seconds int) (Room, error) {
	host, err := uuid.Parse(hostID)
	if err != nil {
		return Room{}, err
	}
	row, err := p.q.InsertRoom(ctx, sqlcdb.InsertRoomParams{
		HostID:       host,
		Passcode:     passcode,
		WinsToFinish: int32(wins),
		TurnSeconds:  int32(seconds),
	})
	if err != nil {
		if isUnique(err) {
			return Room{}, ErrPasscodeTaken
		}
		return Room{}, err
	}
	return roomFromRow(row), nil
}

func (p *Postgres) GetRoom(ctx context.Context, id string) (Room, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return Room{}, ErrNotFound
	}
	row, err := p.q.GetRoom(ctx, uid)
	if err != nil {
		return Room{}, mapNotFound(err)
	}
	return roomFromRow(row), nil
}

func (p *Postgres) GetOpenByPasscode(ctx context.Context, passcode string) (Room, error) {
	row, err := p.q.GetOpenRoomByPasscode(ctx, passcode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Room{}, ErrWrongPasscode
		}
		return Room{}, err
	}
	return roomFromRow(row), nil
}

func (p *Postgres) AddPlayer(ctx context.Context, roomID, userID string) error {
	rid, err := uuid.Parse(roomID)
	if err != nil {
		return ErrNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return p.q.InsertRoomPlayer(ctx, sqlcdb.InsertRoomPlayerParams{RoomID: rid, UserID: uid})
}

func (p *Postgres) RemovePlayer(ctx context.Context, roomID, userID string) error {
	rid, err := uuid.Parse(roomID)
	if err != nil {
		return ErrNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return p.q.DeleteRoomPlayer(ctx, sqlcdb.DeleteRoomPlayerParams{RoomID: rid, UserID: uid})
}

func (p *Postgres) ListPlayers(ctx context.Context, roomID string) ([]Player, error) {
	rid, err := uuid.Parse(roomID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := p.q.ListRoomPlayers(ctx, rid)
	if err != nil {
		return nil, err
	}
	room, err := p.q.GetRoom(ctx, rid)
	if err != nil {
		return nil, mapNotFound(err)
	}
	out := make([]Player, 0, len(rows))
	for _, r := range rows {
		out = append(out, Player{
			ID:          r.ID.String(),
			DisplayName: r.DisplayName,
			Avatar:      ident.DefaultAvatar,
			Color:       ident.DefaultColor,
			IsHost:      r.ID == room.HostID,
			Joined:      true,
		})
	}
	return out, nil
}

func (p *Postgres) CloseRoom(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return ErrNotFound
	}
	return p.q.CloseRoom(ctx, uid)
}

func (p *Postgres) SetHost(ctx context.Context, roomID, hostID string) error {
	rid, err := uuid.Parse(roomID)
	if err != nil {
		return ErrNotFound
	}
	hid, err := uuid.Parse(hostID)
	if err != nil {
		return err
	}
	return p.q.SetRoomHost(ctx, sqlcdb.SetRoomHostParams{ID: rid, HostID: hid})
}

func (p *Postgres) SetStatus(ctx context.Context, id string, status Status) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return ErrNotFound
	}
	return p.q.SetRoomStatus(ctx, sqlcdb.SetRoomStatusParams{
		ID:     uid,
		Status: sqlcdb.RoomStatus(status),
	})
}

func roomFromRow(row sqlcdb.Room) Room {
	return Room{
		ID:           row.ID.String(),
		Passcode:     row.Passcode,
		HostID:       row.HostID.String(),
		WinsToFinish: int(row.WinsToFinish),
		TurnSeconds:  int(row.TurnSeconds),
		Status:       Status(row.Status),
	}
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
