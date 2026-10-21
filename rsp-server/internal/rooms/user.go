package rooms

import (
	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/sqlcdb"
)

func PlayerFromUser(u sqlcdb.User) ident.Player {
	p := ident.Player{
		ID:          u.ID.String(),
		DisplayName: u.DisplayName,
		IsGuest:     u.IsGuest,
		Avatar:      u.Avatar,
		Color:       u.Color,
	}
	if u.Username != nil {
		p.Username = *u.Username
	}
	if u.Email != nil {
		p.Email = *u.Email
	}
	return p.WithDefaults()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
