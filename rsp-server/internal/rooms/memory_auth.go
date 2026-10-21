package rooms

import (
	"context"
	"strings"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/google/uuid"
)

func (m *Memory) CreateGuest(_ context.Context, displayName, avatar, color string) (ident.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := ident.Player{
		ID:          uuid.NewString(),
		DisplayName: displayName,
		Avatar:      avatar,
		Color:       color,
		IsGuest:     true,
	}.WithDefaults()
	m.users[p.ID] = p
	return p, nil
}

func (m *Memory) CreatePasswordUser(_ context.Context, username, hash, email, displayName, avatar, color string) (ident.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.usernameTakenLocked(username, "") {
		return ident.Player{}, ErrUsernameTaken
	}
	if email != "" && m.emailTakenLocked(email, "") {
		return ident.Player{}, ErrEmailTaken
	}
	p := ident.Player{
		ID:          uuid.NewString(),
		DisplayName: displayName,
		Username:    username,
		Email:       email,
		Avatar:      avatar,
		Color:       color,
		IsGuest:     false,
	}.WithDefaults()
	m.users[p.ID] = p
	m.hashes[p.ID] = hash
	return p, nil
}

func (m *Memory) GetPlayer(_ context.Context, id string) (ident.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.users[id]
	if !ok {
		return ident.Player{}, ErrNotFound
	}
	return p, nil
}

func (m *Memory) GetPlayerByUsername(_ context.Context, username string) (ident.Player, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.users {
		if p.Username != "" && strings.EqualFold(p.Username, username) {
			return p, m.hashes[p.ID], nil
		}
	}
	return ident.Player{}, "", ErrNotFound
}

func (m *Memory) GetPlayerByEmail(_ context.Context, email string) (ident.Player, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.users {
		if p.Email != "" && strings.EqualFold(p.Email, email) {
			return p, m.hashes[p.ID], nil
		}
	}
	return ident.Player{}, "", ErrNotFound
}

func (m *Memory) UpgradeGuest(_ context.Context, id, username, hash, email, displayName string) (ident.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.users[id]
	if !ok {
		return ident.Player{}, ErrNotFound
	}
	if !p.IsGuest {
		return ident.Player{}, ErrNotGuest
	}
	if m.usernameTakenLocked(username, id) {
		return ident.Player{}, ErrUsernameTaken
	}
	if email != "" && m.emailTakenLocked(email, id) {
		return ident.Player{}, ErrEmailTaken
	}
	p.Username = username
	p.Email = email
	p.IsGuest = false
	if displayName != "" {
		p.DisplayName = displayName
	}
	m.users[id] = p
	m.hashes[id] = hash
	return p, nil
}

func (m *Memory) SetPassword(_ context.Context, id, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[id]; !ok {
		return ErrNotFound
	}
	m.hashes[id] = hash
	return nil
}

func (m *Memory) LinkProvider(_ context.Context, userID, provider, providerUserID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[provider+"\x00"+providerUserID] = userID
	return nil
}

func (m *Memory) GetByProvider(_ context.Context, provider, providerUserID string) (ident.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.providers[provider+"\x00"+providerUserID]
	if !ok {
		return ident.Player{}, ErrNotFound
	}
	p, ok := m.users[id]
	if !ok {
		return ident.Player{}, ErrNotFound
	}
	return p, nil
}

func (m *Memory) CreateGoogleUser(_ context.Context, email, displayName, providerUserID, avatar, color string) (ident.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if email != "" && m.emailTakenLocked(email, "") {
		return ident.Player{}, ErrEmailTaken
	}
	p := ident.Player{
		ID:          uuid.NewString(),
		DisplayName: displayName,
		Email:       email,
		Avatar:      avatar,
		Color:       color,
		IsGuest:     false,
	}.WithDefaults()
	m.users[p.ID] = p
	m.providers["google\x00"+providerUserID] = p.ID
	return p, nil
}

func (m *Memory) InsertRefresh(_ context.Context, userID, hash string, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refresh[hash] = memRefresh{
		ID:        uuid.NewString(),
		UserID:    userID,
		Hash:      hash,
		ExpiresAt: expires,
		CreatedAt: m.now(),
	}
	return nil
}

func (m *Memory) GetRefresh(_ context.Context, hash string) (userID string, expires, created time.Time, revoked bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.refresh[hash]
	if !ok {
		return "", time.Time{}, time.Time{}, false, ErrNotFound
	}
	return row.UserID, row.ExpiresAt, row.CreatedAt, row.Revoked, nil
}

func (m *Memory) RevokeRefresh(_ context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.refresh[hash]
	if !ok {
		return ErrNotFound
	}
	row.Revoked = true
	m.refresh[hash] = row
	return nil
}

func (m *Memory) RevokeAllRefresh(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, row := range m.refresh {
		if row.UserID == userID {
			row.Revoked = true
			m.refresh[k] = row
		}
	}
	return nil
}

func (m *Memory) InsertReset(_ context.Context, userID, hash string, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resets[hash] = memReset{
		ID:        uuid.NewString(),
		UserID:    userID,
		Hash:      hash,
		ExpiresAt: expires,
	}
	return nil
}

func (m *Memory) ConsumeReset(_ context.Context, hash string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.resets[hash]
	if !ok || row.Used || !m.now().Before(row.ExpiresAt) {
		return "", ErrNotFound
	}
	row.Used = true
	m.resets[hash] = row
	return row.UserID, nil
}

func (m *Memory) usernameTakenLocked(username, exceptID string) bool {
	for _, p := range m.users {
		if p.ID != exceptID && p.Username != "" && strings.EqualFold(p.Username, username) {
			return true
		}
	}
	return false
}

func (m *Memory) emailTakenLocked(email, exceptID string) bool {
	for _, p := range m.users {
		if p.ID != exceptID && p.Email != "" && strings.EqualFold(p.Email, email) {
			return true
		}
	}
	return false
}
