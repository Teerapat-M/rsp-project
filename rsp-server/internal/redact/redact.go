// Package redact strips secrets from strings that may be logged or returned
// in errors. Third-party libraries (pgx, golang-migrate) often embed the
// full Postgres DSN, including the password.
package redact

import (
	"errors"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
)

const mask = "***"

var secretEnv = []string{
	"DATABASE_URL",
	"DATABASE_URL_MIGRATE",
	"JWT_SECRET",
	"RESEND_API_KEY",
	"SMTP_PASSWORD",
}

var (
	// postgres://user:pass@host and other URLs with userinfo.
	urlUserinfo = regexp.MustCompile(`(?i)((?:postgres(?:ql)?|https?|wss?|redis|amqp|mongodb)://)[^@\s"'<>]+@`)
	kvPassword  = regexp.MustCompile("(?i)(\\bpassword=)[^\\s;'\"`]+")
	kvUser      = regexp.MustCompile("(?i)(\\buser=)[^\\s;'\"`]+")
	querySecret = regexp.MustCompile(`(?i)([?&](?:token|access_token|refresh_token|id_token|password|secret)=)[^&\s"'<>]*`)
	bearer      = regexp.MustCompile(`(?i)(bearer\s+)\S+`)
	jsonSecret  = regexp.MustCompile(`(?i)("(?:accessToken|refreshToken|idToken|password|token)"\s*:\s*")[^"]*(")`)
	mapSecret   = regexp.MustCompile(`(?i)(\b(?:accessToken|refreshToken|idToken)[:])[^\s\]]+`)
)

// String redacts userinfo, passwords, tokens, and known secret env values.
func String(s string) string {
	if s == "" {
		return s
	}
	for _, key := range secretEnv {
		if v := os.Getenv(key); v != "" {
			s = strings.ReplaceAll(s, v, mask)
		}
	}
	s = urlUserinfo.ReplaceAllString(s, "${1}"+mask+"@")
	s = kvPassword.ReplaceAllString(s, "${1}"+mask)
	s = kvUser.ReplaceAllString(s, "${1}"+mask)
	s = querySecret.ReplaceAllString(s, "${1}"+mask)
	s = bearer.ReplaceAllString(s, "${1}"+mask)
	s = jsonSecret.ReplaceAllString(s, "${1}"+mask+"${2}")
	s = mapSecret.ReplaceAllString(s, "${1}"+mask)
	return s
}

// Error returns a new error whose message is String(err.Error()).
// It does not unwrap to the original, so the DSN cannot leak via errors.Unwrap.
func Error(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(String(err.Error()))
}

// DSNError redacts err and also strips userinfo taken from dsn (pgx ping
// errors look like `user=alice database=postgres` with no password, but
// the username is still userinfo).
func DSNError(dsn string, err error) error {
	if err == nil {
		return nil
	}
	msg := String(err.Error())
	if dsn != "" {
		msg = strings.ReplaceAll(msg, dsn, mask)
		if u, perr := url.Parse(dsn); perr == nil && u.User != nil {
			if user := u.User.Username(); user != "" {
				msg = strings.ReplaceAll(msg, user, mask)
			}
			if pass, ok := u.User.Password(); ok && pass != "" {
				msg = strings.ReplaceAll(msg, pass, mask)
			}
		}
	}
	return errors.New(msg)
}

type writer struct{ dst io.Writer }

// Writer wraps dst so every Write is passed through String.
func Writer(dst io.Writer) io.Writer {
	if dst == nil {
		dst = io.Discard
	}
	return writer{dst: dst}
}

func (w writer) Write(p []byte) (int, error) {
	_, err := io.WriteString(w.dst, String(string(p)))
	return len(p), err
}
