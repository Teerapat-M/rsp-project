package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/auth"
	"github.com/Teerapat-M/rsp-project/internal/hub"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthRegisterLoginRefreshLogout(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	res, err := http.Post(ts.URL+"/auth/register", "application/json", bytes.NewBufferString(`{"username":"bell_ok","password":"secret1","email":"bell@example.com"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("register %d", res.StatusCode)
	}
	var sess struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			IsGuest  bool   `json:"isGuest"`
		} `json:"user"`
	}
	decode(t, res, &sess)
	if sess.AccessToken == "" || sess.RefreshToken == "" || sess.User.IsGuest {
		t.Fatalf("%+v", sess)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("me %d", res.StatusCode)
	}
	decode(t, res, nil)

	res, err = http.Post(ts.URL+"/auth/login", "application/json", bytes.NewBufferString(`{"username":"bell_ok","password":"secret1","rememberMe":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login %d", res.StatusCode)
	}
	decode(t, res, &sess)

	res, err = http.Post(ts.URL+"/auth/refresh", "application/json", bytes.NewBufferString(`{"refreshToken":"`+sess.RefreshToken+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("refresh %d", res.StatusCode)
	}
	decode(t, res, &sess)

	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/auth/logout", bytes.NewBufferString(`{"refreshToken":"`+sess.RefreshToken+`"}`))
	req.Header.Set("Authorization", "Bearer "+sess.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("logout %d", res.StatusCode)
	}
	decode(t, res, nil)
}

func TestDevGuestReturnsJWT(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()
	res, err := http.Post(ts.URL+"/dev/guest", "application/json", bytes.NewBufferString(`{"displayName":"Mina","avatar":"paper"}`))
	if err != nil {
		t.Fatal(err)
	}
	var sess struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			IsGuest     bool   `json:"isGuest"`
		} `json:"user"`
	}
	decode(t, res, &sess)
	if sess.AccessToken == sess.User.ID {
		t.Fatal("dev guest token should be a JWT, not the raw user id")
	}
	if !sess.User.IsGuest || sess.User.DisplayName != "Mina" {
		t.Fatalf("%+v", sess)
	}
}

func TestDevGuestHiddenInProduction(t *testing.T) {
	st := rooms.NewMemory()
	h := hub.New(st, hub.Config{Grace: time.Second})
	a := auth.New(st, auth.Config{
		Secret:     []byte("test-secret-at-least-32-bytes-long!!"),
		AppEnv:     "production",
		BcryptCost: bcrypt.MinCost,
	})
	ts := httptestServer(t, h, a)
	defer ts.Close()
	res, err := http.Post(ts.URL+"/dev/guest", "application/json", bytes.NewBufferString(`{"displayName":"Mina"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", res.StatusCode)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	res = doJSON(t, ts, http.MethodPost, "/rooms", "alice", map[string]int{"winsToFinish": 1, "turnSeconds": 10})
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("X-Player-ID should not work in production, got %d", res.StatusCode)
	}
	decode(t, res, nil)
}

func TestGoogleUnconfigured(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()
	res, err := http.Post(ts.URL+"/auth/google", "application/json", bytes.NewBufferString(`{"idToken":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d", res.StatusCode)
	}
	decode(t, res, nil)
}

func httptestServer(t *testing.T, h *hub.Hub, a *auth.Service) *httptest.Server {
	t.Helper()
	return httptest.NewServer(New(h, a).Router())
}
