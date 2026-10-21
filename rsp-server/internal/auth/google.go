package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GoogleIdentity is the verified Google account.
type GoogleIdentity struct {
	Subject string
	Email   string
	Name    string
}

// GoogleVerifier checks an ID token. Tests stub this; production hits tokeninfo.
type GoogleVerifier interface {
	Verify(ctx context.Context, idToken string) (GoogleIdentity, error)
}

// TokenInfoVerifier calls Google's tokeninfo endpoint and checks aud == clientID.
// No client secret. If clientID is empty, Verify returns ErrGoogleDisabled.
type TokenInfoVerifier struct {
	ClientID string
	HTTP     *http.Client
}

func (v TokenInfoVerifier) Verify(ctx context.Context, idToken string) (GoogleIdentity, error) {
	if strings.TrimSpace(v.ClientID) == "" {
		return GoogleIdentity{}, ErrGoogleDisabled
	}
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return GoogleIdentity{}, ErrGoogleToken
	}
	client := v.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	u := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return GoogleIdentity{}, err
	}
	res, err := client.Do(req)
	if err != nil {
		return GoogleIdentity{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return GoogleIdentity{}, ErrGoogleToken
	}
	var payload struct {
		Aud           string `json:"aud"`
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return GoogleIdentity{}, ErrGoogleToken
	}
	if payload.Aud != v.ClientID || payload.Sub == "" {
		return GoogleIdentity{}, ErrGoogleToken
	}
	if payload.EmailVerified == "false" {
		return GoogleIdentity{}, fmt.Errorf("%w: email not verified", ErrGoogleToken)
	}
	name := payload.Name
	if name == "" {
		name = payload.Email
	}
	if name == "" {
		name = "Player"
	}
	return GoogleIdentity{Subject: payload.Sub, Email: payload.Email, Name: name}, nil
}
