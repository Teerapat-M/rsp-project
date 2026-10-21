package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/ident"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"golang.org/x/crypto/bcrypt"
)

var usernameRE = regexp.MustCompile(`^[A-Za-z0-9_]{3,20}$`)

const (
	minPassword = 8
	maxPassword = 72
	// DevJWTSecret is the documented fallback when JWT_SECRET is unset in
	// development. Production must set a real secret (see cmd/server).
	DevJWTSecret = "dev-only-change-me-please-32b!!"
)

type Config struct {
	Secret          []byte
	AccessTTL       time.Duration
	RefreshTTL      time.Duration
	RefreshRemember time.Duration
	ResetTTL        time.Duration
	BcryptCost      int
	AppEnv          string
	PublicBase      string // e.g. http://localhost:5173 for reset links
	Mail            Sender
	Google          GoogleVerifier
	Now             func() time.Time
}

type Service struct {
	users           Users
	secret          []byte
	accessTTL       time.Duration
	refreshTTL      time.Duration
	refreshRemember time.Duration
	resetTTL        time.Duration
	cost            int
	appEnv          string
	publicBase      string
	mail            Sender
	google          GoogleVerifier
	now             func() time.Time
}

func New(users Users, cfg Config) *Service {
	if cfg.AccessTTL <= 0 {
		cfg.AccessTTL = 15 * time.Minute
	}
	if cfg.RefreshTTL <= 0 {
		cfg.RefreshTTL = 7 * 24 * time.Hour
	}
	if cfg.RefreshRemember <= 0 {
		cfg.RefreshRemember = 30 * 24 * time.Hour
	}
	if cfg.ResetTTL <= 0 {
		cfg.ResetTTL = time.Hour
	}
	if cfg.BcryptCost < bcrypt.MinCost {
		cfg.BcryptCost = 12
	}
	if cfg.BcryptCost < 10 && !isDev(cfg.AppEnv) {
		cfg.BcryptCost = 10
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Mail == nil {
		cfg.Mail = LogSender{}
	}
	if len(cfg.Secret) == 0 {
		cfg.Secret = []byte(DevJWTSecret)
	}
	return &Service{
		users:           users,
		secret:          cfg.Secret,
		accessTTL:       cfg.AccessTTL,
		refreshTTL:      cfg.RefreshTTL,
		refreshRemember: cfg.RefreshRemember,
		resetTTL:        cfg.ResetTTL,
		cost:            cfg.BcryptCost,
		appEnv:          cfg.AppEnv,
		publicBase:      strings.TrimRight(cfg.PublicBase, "/"),
		mail:            cfg.Mail,
		google:          cfg.Google,
		now:             cfg.Now,
	}
}

func (s *Service) Dev() bool { return isDev(s.appEnv) }

func isDev(env string) bool {
	e := strings.ToLower(strings.TrimSpace(env))
	return e == "" || e == "dev" || e == "development" || e == "test"
}

type Session struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	User         ident.Player `json:"user"`
}

type UserJSON struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Username    string `json:"username,omitempty"`
	Email       string `json:"email,omitempty"`
	IsGuest     bool   `json:"isGuest"`
	Avatar      string `json:"avatar"`
	Color       string `json:"color"`
}

func PublicUser(p ident.Player) UserJSON {
	p = p.WithDefaults()
	return UserJSON{
		ID:          p.ID,
		DisplayName: p.DisplayName,
		Username:    p.Username,
		Email:       p.Email,
		IsGuest:     p.IsGuest,
		Avatar:      p.Avatar,
		Color:       p.Color,
	}
}

func (s *Service) issue(ctx context.Context, p ident.Player, remember bool) (Session, error) {
	access, err := s.signAccess(p.ID)
	if err != nil {
		return Session{}, err
	}
	raw, err := randomToken()
	if err != nil {
		return Session{}, err
	}
	ttl := s.ttlForRemember(remember)
	if err := s.users.InsertRefresh(ctx, p.ID, hashToken(raw), s.now().Add(ttl)); err != nil {
		return Session{}, err
	}
	return Session{AccessToken: access, RefreshToken: raw, User: p.WithDefaults()}, nil
}

func (s *Service) Guest(ctx context.Context, displayName, avatar, color string) (Session, error) {
	name := strings.TrimSpace(displayName)
	if len(name) < 3 || len(name) > 32 {
		return Session{}, fmt.Errorf("%w: display name must be 3–32 characters", ErrValidation)
	}
	p, err := s.users.CreateGuest(ctx, name, avatar, color)
	if err != nil {
		return Session{}, err
	}
	return s.issue(ctx, p, false)
}

func (s *Service) Register(ctx context.Context, username, password, email string) (Session, error) {
	username, err := normalizeUsername(username)
	if err != nil {
		return Session{}, err
	}
	if err := validatePassword(password); err != nil {
		return Session{}, err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return Session{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return Session{}, err
	}
	p, err := s.users.CreatePasswordUser(ctx, username, string(hash), email, username, ident.DefaultAvatar, ident.DefaultColor)
	if err != nil {
		return Session{}, mapStore(err)
	}
	return s.issue(ctx, p, false)
}

func (s *Service) Login(ctx context.Context, username, password string, remember bool) (Session, error) {
	p, hash, err := s.users.GetPlayerByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return Session{}, ErrUnauthorized
	}
	if hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return Session{}, ErrUnauthorized
	}
	return s.issue(ctx, p, remember)
}

func (s *Service) Refresh(ctx context.Context, raw string) (Session, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Session{}, ErrUnauthorized
	}
	h := hashToken(raw)
	userID, expires, created, revoked, err := s.users.GetRefresh(ctx, h)
	if err != nil {
		return Session{}, ErrUnauthorized
	}
	if revoked {
		_ = s.users.RevokeAllRefresh(ctx, userID)
		return Session{}, ErrUnauthorized
	}
	if !s.now().Before(expires) {
		return Session{}, ErrUnauthorized
	}
	_ = s.users.RevokeRefresh(ctx, h)
	p, err := s.users.GetPlayer(ctx, userID)
	if err != nil {
		return Session{}, ErrUnauthorized
	}
	ttl := expires.Sub(created)
	if ttl <= 0 {
		ttl = s.refreshTTL
	}
	access, err := s.signAccess(p.ID)
	if err != nil {
		return Session{}, err
	}
	next, err := randomToken()
	if err != nil {
		return Session{}, err
	}
	if err := s.users.InsertRefresh(ctx, p.ID, hashToken(next), s.now().Add(ttl)); err != nil {
		return Session{}, err
	}
	return Session{AccessToken: access, RefreshToken: next, User: p.WithDefaults()}, nil
}

func (s *Service) Logout(ctx context.Context, userID, refreshRaw string) error {
	if refreshRaw != "" {
		return s.users.RevokeRefresh(ctx, hashToken(refreshRaw))
	}
	return s.users.RevokeAllRefresh(ctx, userID)
}

func (s *Service) Me(ctx context.Context, userID string) (ident.Player, error) {
	return s.users.GetPlayer(ctx, userID)
}

func (s *Service) ParseAccess(raw string) (string, error) {
	return s.parseAccess(raw)
}

func (s *Service) Upgrade(ctx context.Context, userID, username, password, email string) (Session, error) {
	username, err := normalizeUsername(username)
	if err != nil {
		return Session{}, err
	}
	if err := validatePassword(password); err != nil {
		return Session{}, err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return Session{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return Session{}, err
	}
	p, err := s.users.UpgradeGuest(ctx, userID, username, string(hash), email, username)
	if err != nil {
		return Session{}, mapStore(err)
	}
	return s.issue(ctx, p, false)
}

func (s *Service) Forgot(ctx context.Context, email string) (resetURL string, err error) {
	email, err = normalizeEmail(email)
	if err != nil || email == "" {
		return "", nil // same response; invalid email is "not found"
	}
	p, hash, err := s.users.GetPlayerByEmail(ctx, email)
	if err != nil || hash == "" {
		return "", nil
	}
	raw, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := s.users.InsertReset(ctx, p.ID, hashToken(raw), s.now().Add(s.resetTTL)); err != nil {
		return "", err
	}
	base := s.publicBase
	if base == "" {
		base = "http://localhost:5173"
	}
	u := base + "/reset?token=" + raw
	if err := s.mail.SendPasswordReset(ctx, email, u); err != nil {
		return "", err
	}
	return u, nil
}

func (s *Service) Reset(ctx context.Context, rawToken, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	userID, err := s.users.ConsumeReset(ctx, hashToken(strings.TrimSpace(rawToken)))
	if err != nil {
		return ErrResetInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return err
	}
	return s.users.SetPassword(ctx, userID, string(hash))
}

func (s *Service) Google(ctx context.Context, idToken string) (Session, error) {
	if s.google == nil {
		return Session{}, ErrGoogleDisabled
	}
	identities, err := s.google.Verify(ctx, idToken)
	if err != nil {
		if errors.Is(err, ErrGoogleDisabled) {
			return Session{}, err
		}
		return Session{}, ErrGoogleToken
	}
	if p, err := s.users.GetByProvider(ctx, "google", identities.Subject); err == nil {
		return s.issue(ctx, p, false)
	}
	// Do not auto-link Google onto an existing password user just because
	// the emails match — that is an account-takeover if the password
	// address was never proven. Linking is an explicit later step.
	p, err := s.users.CreateGoogleUser(ctx, identities.Email, identities.Name, identities.Subject, ident.DefaultAvatar, ident.DefaultColor)
	if err != nil {
		return Session{}, mapStore(err)
	}
	return s.issue(ctx, p, false)
}

func normalizeUsername(raw string) (string, error) {
	u := strings.TrimSpace(raw)
	if !usernameRE.MatchString(u) {
		return "", fmt.Errorf("%w: username must be 3–20 letters, numbers, or _", ErrValidation)
	}
	return u, nil
}

func validatePassword(p string) error {
	if len(p) < minPassword {
		return fmt.Errorf("%w: password must be at least %d characters", ErrValidation, minPassword)
	}
	if len(p) > maxPassword {
		return fmt.Errorf("%w: password is too long", ErrValidation)
	}
	return nil
}

func normalizeEmail(raw string) (string, error) {
	e := strings.TrimSpace(raw)
	if e == "" {
		return "", nil
	}
	addr, err := mail.ParseAddress(e)
	if err != nil || addr.Address == "" {
		return "", fmt.Errorf("%w: invalid email", ErrValidation)
	}
	return strings.ToLower(addr.Address), nil
}

var (
	_ Users = (*rooms.Memory)(nil)
	_ Users = (*rooms.Postgres)(nil)
)

func mapStore(err error) error {
	switch {
	case errors.Is(err, rooms.ErrUsernameTaken):
		return ErrUsernameTaken
	case errors.Is(err, rooms.ErrEmailTaken):
		return ErrEmailTaken
	case errors.Is(err, rooms.ErrNotGuest):
		return ErrNotGuest
	default:
		return err
	}
}
