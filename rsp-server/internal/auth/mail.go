package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/Teerapat-M/rsp-project/internal/redact"
)

// Sender delivers a password-reset link. Dev uses LogSender.
type Sender interface {
	SendPasswordReset(ctx context.Context, to, resetURL string) error
}

// LogSender prints the reset URL. Used when EMAIL_PROVIDER=log (the default).
type LogSender struct{}

func (LogSender) SendPasswordReset(_ context.Context, to, resetURL string) error {
	log.Printf("password reset for %s: %s", to, resetURL)
	return nil
}

// ResendSender posts to Resend's emails API. Needs RESEND_API_KEY and EMAIL_FROM.
type ResendSender struct {
	APIKey string
	From   string
	HTTP   *http.Client
}

func (s ResendSender) SendPasswordReset(ctx context.Context, to, resetURL string) error {
	if s.APIKey == "" || s.From == "" {
		return fmt.Errorf("resend: missing API key or from address")
	}
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	body, err := json.Marshal(map[string]any{
		"from":    s.From,
		"to":      []string{to},
		"subject": "Reset your RSP password",
		"text":    "Reset your password: " + resetURL,
		"html":    `<p>Reset your password:</p><p><a href="` + resetURL + `">` + resetURL + `</a></p>`,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return redact.Error(err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("resend: status %d", res.StatusCode)
	}
	return nil
}

// SMTPSender sends via net/smtp. Needs SMTP_HOST, SMTP_FROM; user/pass optional.
type SMTPSender struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func (s SMTPSender) SendPasswordReset(_ context.Context, to, resetURL string) error {
	if s.Host == "" || s.From == "" {
		return fmt.Errorf("smtp: missing host or from address")
	}
	port := s.Port
	if port == "" {
		port = "587"
	}
	addr := s.Host + ":" + port
	msg := strings.Join([]string{
		"From: " + s.From,
		"To: " + to,
		"Subject: Reset your RSP password",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Reset your password: " + resetURL,
	}, "\r\n")
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Password, s.Host)
	}
	if err := smtp.SendMail(addr, auth, s.From, []string{to}, []byte(msg)); err != nil {
		return redact.Error(err)
	}
	return nil
}

// SenderFromEnv picks log / resend / smtp. Unknown values fall back to log.
func SenderFromEnv(provider, resendKey, from, smtpHost, smtpPort, smtpUser, smtpPass string) Sender {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "resend":
		return ResendSender{APIKey: resendKey, From: from}
	case "smtp":
		return SMTPSender{Host: smtpHost, Port: smtpPort, User: smtpUser, Password: smtpPass, From: from}
	default:
		return LogSender{}
	}
}
