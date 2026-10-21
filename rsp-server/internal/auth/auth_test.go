package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"golang.org/x/crypto/bcrypt"
)

func testSvc(t *testing.T) (*Service, *rooms.Memory) {
	t.Helper()
	st := rooms.NewMemory()
	s := New(st, Config{
		Secret:          []byte("test-secret-at-least-32-bytes-long!!"),
		AppEnv:          "test",
		BcryptCost:      bcrypt.MinCost,
		AccessTTL:       time.Hour,
		RefreshTTL:      24 * time.Hour,
		RefreshRemember: 72 * time.Hour,
		ResetTTL:        time.Hour,
		PublicBase:      "http://localhost:5173",
	})
	return s, st
}

func TestRegisterLoginAndMe(t *testing.T) {
	t.Parallel()
	s, _ := testSvc(t)
	ctx := context.Background()
	sess, err := s.Register(ctx, "Player_1", "password", "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if sess.AccessToken == "" || sess.RefreshToken == "" || sess.User.IsGuest || sess.User.Username != "Player_1" {
		t.Fatalf("%+v", sess)
	}
	if _, err := s.Register(ctx, "player_1", "password", ""); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("case-insensitive username: %v", err)
	}
	uid, err := s.ParseAccess(sess.AccessToken)
	if err != nil || uid != sess.User.ID {
		t.Fatalf("parse %s %v", uid, err)
	}
	me, err := s.Me(ctx, uid)
	if err != nil || me.Email != "a@example.com" {
		t.Fatalf("me %+v %v", me, err)
	}
	if _, err := s.Login(ctx, "Player_1", "wrong", false); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("bad pass: %v", err)
	}
	again, err := s.Login(ctx, "player_1", "password", false)
	if err != nil || again.User.ID != sess.User.ID {
		t.Fatalf("login %+v %v", again, err)
	}
}

func TestRememberMeLongerRefresh(t *testing.T) {
	t.Parallel()
	s, st := testSvc(t)
	ctx := context.Background()
	_, _ = s.Register(ctx, "rem_user", "password", "")
	short, err := s.Login(ctx, "rem_user", "password", false)
	if err != nil {
		t.Fatal(err)
	}
	long, err := s.Login(ctx, "rem_user", "password", true)
	if err != nil {
		t.Fatal(err)
	}
	_, expShort, _, _, err := st.GetRefresh(ctx, sha(short.RefreshToken))
	if err != nil {
		t.Fatal(err)
	}
	_, expLong, _, _, err := st.GetRefresh(ctx, sha(long.RefreshToken))
	if err != nil {
		t.Fatal(err)
	}
	if expLong.Sub(expShort) < 40*time.Hour {
		t.Fatalf("remember %s vs %s", expLong, expShort)
	}
}

func TestRefreshRotationAndReuse(t *testing.T) {
	t.Parallel()
	s, _ := testSvc(t)
	ctx := context.Background()
	sess, err := s.Register(ctx, "rotator", "password", "")
	if err != nil {
		t.Fatal(err)
	}
	next, err := s.Refresh(ctx, sess.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if next.RefreshToken == sess.RefreshToken {
		t.Fatal("should rotate the refresh token")
	}
	if _, err := s.Refresh(ctx, sess.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("reuse: %v", err)
	}
	if _, err := s.Refresh(ctx, next.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("reuse should revoke the family: %v", err)
	}
}

func TestGuestUpgradeKeepsID(t *testing.T) {
	t.Parallel()
	s, _ := testSvc(t)
	ctx := context.Background()
	guest, err := s.Guest(ctx, "Mina", "paper", "#93BFE3")
	if err != nil {
		t.Fatal(err)
	}
	if !guest.User.IsGuest || guest.User.Avatar != "paper" {
		t.Fatalf("%+v", guest.User)
	}
	up, err := s.Upgrade(ctx, guest.User.ID, "mina_ok", "password", "mina@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if up.User.ID != guest.User.ID {
		t.Fatalf("id changed %s → %s", guest.User.ID, up.User.ID)
	}
	if up.User.IsGuest || up.User.Username != "mina_ok" {
		t.Fatalf("%+v", up.User)
	}
	if _, err := s.Upgrade(ctx, up.User.ID, "other", "password", ""); !errors.Is(err, ErrNotGuest) {
		t.Fatalf("second upgrade: %v", err)
	}
}

func TestForgotResetNoEnumeration(t *testing.T) {
	t.Parallel()
	mail := &memMail{}
	st := rooms.NewMemory()
	s := New(st, Config{
		Secret:     []byte("test-secret-at-least-32-bytes-long!!"),
		AppEnv:     "test",
		BcryptCost: bcrypt.MinCost,
		Mail:       mail,
		PublicBase: "http://localhost:5173",
		ResetTTL:   time.Hour,
	})
	ctx := context.Background()
	_, _ = s.Register(ctx, "resetme", "password", "me@example.com")

	u1, err := s.Forgot(ctx, "nobody@example.com")
	if err != nil || u1 != "" || mail.lastURL != "" {
		t.Fatalf("unknown email should be silent: url=%q err=%v mail=%+v", u1, err, mail)
	}
	u2, err := s.Forgot(ctx, "me@example.com")
	if err != nil || u2 == "" || !strings.Contains(mail.lastURL, "/reset?token=") {
		t.Fatalf("known email: url=%q err=%v mail=%+v", u2, err, mail)
	}
	token := mail.lastURL[strings.LastIndex(mail.lastURL, "=")+1:]
	if err := s.Reset(ctx, token, "newpass1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Login(ctx, "resetme", "password", false); !errors.Is(err, ErrUnauthorized) {
		t.Fatal("old password should fail")
	}
	if _, err := s.Login(ctx, "resetme", "newpass1", false); err != nil {
		t.Fatal(err)
	}
	if err := s.Reset(ctx, token, "another1"); !errors.Is(err, ErrResetInvalid) {
		t.Fatalf("reuse reset: %v", err)
	}
}

func TestGoogleDisabledAndStub(t *testing.T) {
	t.Parallel()
	s, _ := testSvc(t)
	ctx := context.Background()
	if _, err := s.Google(ctx, "tok"); !errors.Is(err, ErrGoogleDisabled) {
		t.Fatalf("no verifier: %v", err)
	}
	s.google = stubGoogle{id: GoogleIdentity{Subject: "g-1", Email: "g@example.com", Name: "Goo"}, err: nil}
	sess, err := s.Google(ctx, "id-token")
	if err != nil || sess.User.IsGuest || sess.User.Email != "g@example.com" {
		t.Fatalf("%+v %v", sess, err)
	}
	again, err := s.Google(ctx, "id-token")
	if err != nil || again.User.ID != sess.User.ID {
		t.Fatalf("same google account should reuse users.id: %s vs %s", again.User.ID, sess.User.ID)
	}
}

func TestGoogleDoesNotLinkExistingEmail(t *testing.T) {
	t.Parallel()
	s, _ := testSvc(t)
	ctx := context.Background()
	sess, err := s.Register(ctx, "pw_user", "password", "same@example.com")
	if err != nil {
		t.Fatal(err)
	}
	s.google = stubGoogle{id: GoogleIdentity{Subject: "g-99", Email: "same@example.com", Name: "Goo"}}
	if _, err := s.Google(ctx, "id-token"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("google must not take over a password account: %v", err)
	}
	again, err := s.Login(ctx, "pw_user", "password", false)
	if err != nil || again.User.ID != sess.User.ID {
		t.Fatalf("password login after google attempt: %+v %v", again, err)
	}
}

func TestRegisterValidation(t *testing.T) {
	t.Parallel()
	s, _ := testSvc(t)
	ctx := context.Background()
	if _, err := s.Register(ctx, "ab", "password", ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("short username: %v", err)
	}
	if _, err := s.Register(ctx, "good_name", "123", ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("short password: %v", err)
	}
	if _, err := s.Register(ctx, "good_name", "secret1", ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("7-char password: %v", err)
	}
	if _, err := s.Register(ctx, "good_name", "password", "not-an-email"); !errors.Is(err, ErrValidation) {
		t.Fatalf("email: %v", err)
	}
}

func sha(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

type memMail struct{ lastTo, lastURL string }

func (m *memMail) SendPasswordReset(_ context.Context, to, resetURL string) error {
	m.lastTo, m.lastURL = to, resetURL
	return nil
}

type stubGoogle struct {
	id  GoogleIdentity
	err error
}

func (s stubGoogle) Verify(context.Context, string) (GoogleIdentity, error) {
	return s.id, s.err
}
