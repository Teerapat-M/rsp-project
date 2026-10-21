package redact

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const leaked = "leaked-password-xyz"

func TestStringRedactsPostgresUserinfo(t *testing.T) {
	in := `error: failed to open database, "postgres://postgres.abc:` + leaked + `@aws-0-us-east-1.pooler.supabase.com:6543/postgres?sslmode=require": failed to ping`
	got := String(in)
	if strings.Contains(got, leaked) {
		t.Fatalf("password leaked: %s", got)
	}
	if !strings.Contains(got, "postgres://"+mask+"@") {
		t.Fatalf("expected masked userinfo, got %s", got)
	}
	if !strings.Contains(got, "pooler.supabase.com") {
		t.Fatalf("host should remain: %s", got)
	}
}

func TestStringRedactsLibpqPassword(t *testing.T) {
	got := String("host=db.example user=postgres password=" + leaked + " dbname=postgres")
	if strings.Contains(got, leaked) {
		t.Fatalf("password leaked: %s", got)
	}
	if strings.Contains(got, "user=postgres") {
		t.Fatalf("user leaked: %s", got)
	}
	if !strings.Contains(got, "password="+mask) {
		t.Fatalf("expected masked password=, got %s", got)
	}
}

func TestStringRedactsTokensAndEnvSecrets(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt-secret-value-32-bytes-long!!")
	t.Setenv("RESEND_API_KEY", "re_live_secretkey")
	t.Setenv("SMTP_PASSWORD", "smtp-pass-value")
	t.Setenv("DATABASE_URL", "postgres://u:"+leaked+"@db.example:5432/postgres")

	in := strings.Join([]string{
		"JWT jwt-secret-value-32-bytes-long!!",
		"key re_live_secretkey smtp smtp-pass-value",
		`{"accessToken":"eyJhbGciOi.aaa.bbb","password":"nopenope"}`,
		"Authorization: Bearer eyJhbGciOi.aaa.bbb",
		"GET /ws?token=eyJhbGciOi.aaa.bbb",
		"map[accessToken:eyJhbGciOi.aaa.bbb]",
		"postgres://u:" + leaked + "@db.example:5432/postgres",
	}, " | ")
	got := String(in)
	for _, secret := range []string{
		"jwt-secret-value-32-bytes-long!!",
		"re_live_secretkey",
		"smtp-pass-value",
		leaked,
		"eyJhbGciOi.aaa.bbb",
		"nopenope",
	} {
		if strings.Contains(got, secret) {
			t.Fatalf("leaked %q in %s", secret, got)
		}
	}
}

func TestDSNErrorStripsParsedUserinfo(t *testing.T) {
	dsn := "postgres://leakuser:" + leaked + "@127.0.0.1:1/postgres?sslmode=disable"
	orig := errors.New("failed to connect to `user=leakuser database=postgres`: dial tcp 127.0.0.1:1: connect: connection refused")
	got := DSNError(dsn, orig)
	if got == nil {
		t.Fatal("nil")
	}
	msg := got.Error()
	if strings.Contains(msg, leaked) || strings.Contains(msg, "leakuser") {
		t.Fatalf("userinfo leaked: %s", msg)
	}
}

func TestErrorDoesNotUnwrapDSN(t *testing.T) {
	orig := errors.New(`cannot parse "postgres://u:` + leaked + `@localhost/db"`)
	got := Error(orig)
	if got == nil {
		t.Fatal("nil")
	}
	if strings.Contains(got.Error(), leaked) {
		t.Fatalf("password leaked: %s", got)
	}
	if errors.Is(got, orig) {
		t.Fatal("must not unwrap to the original DSN-bearing error")
	}
	if Error(nil) != nil {
		t.Fatal("nil in")
	}
}

func TestWriterRedacts(t *testing.T) {
	var buf bytes.Buffer
	n, err := Writer(&buf).Write([]byte(`postgres://u:` + leaked + `@host/db`))
	if err != nil {
		t.Fatal(err)
	}
	if n != len(`postgres://u:`+leaked+`@host/db`) {
		t.Fatalf("Write n=%d", n)
	}
	if strings.Contains(buf.String(), leaked) {
		t.Fatalf("password leaked: %s", buf.String())
	}
}
