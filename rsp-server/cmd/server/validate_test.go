package main

import (
	"strings"
	"testing"

	"github.com/Teerapat-M/rsp-project/internal/auth"
	"github.com/gin-gonic/gin"
)

func TestListenAddrPrefersHTTPAddrThenPORT(t *testing.T) {
	if got := listenAddr(":8080", "10000"); got != ":8080" {
		t.Fatalf("HTTP_ADDR wins: %s", got)
	}
	if got := listenAddr("", "10000"); got != ":10000" {
		t.Fatalf("Render PORT: %s", got)
	}
	if got := listenAddr("  ", ":9999"); got != ":9999" {
		t.Fatalf("PORT with colon: %s", got)
	}
	if got := listenAddr("", ""); got != ":8080" {
		t.Fatalf("default: %s", got)
	}
}

func TestApplyGinModeProduction(t *testing.T) {
	prev := gin.Mode()
	t.Cleanup(func() { gin.SetMode(prev) })

	gin.SetMode(gin.DebugMode)
	applyGinMode("dev")
	if gin.Mode() != gin.DebugMode {
		t.Fatalf("dev left mode %s, want debug", gin.Mode())
	}
	applyGinMode("production")
	if gin.Mode() != gin.ReleaseMode {
		t.Fatalf("production mode %s, want release", gin.Mode())
	}
	gin.SetMode(gin.DebugMode)
	applyGinMode("prod")
	if gin.Mode() != gin.ReleaseMode {
		t.Fatalf("prod alias mode %s, want release", gin.Mode())
	}
}

func TestValidateBootDevAllowsDefaults(t *testing.T) {
	if err := validateBoot("dev", "", "log", "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestValidateBootProduction(t *testing.T) {
	ok := func() error {
		return validateBoot(
			"production",
			"a-real-secret-at-least-32-bytes-ok!!",
			"resend",
			"https://play.example.com",
			"postgres://rsp:secret@db:5432/rsp?sslmode=disable",
		)
	}
	if err := ok(); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		env  string
		jwt  string
		mail string
		cors string
		dsn  string
		want string
	}{
		{"missing jwt", "production", "", "resend", "https://x.example", "postgres://x", "JWT_SECRET"},
		{"dev jwt", "production", auth.DevJWTSecret, "resend", "https://x.example", "postgres://x", "dev default"},
		{"log mail", "production", "a-real-secret-at-least-32-bytes-ok!!", "log", "https://x.example", "postgres://x", "EMAIL_PROVIDER"},
		{"empty mail", "production", "a-real-secret-at-least-32-bytes-ok!!", "", "https://x.example", "postgres://x", "EMAIL_PROVIDER"},
		{"missing cors", "production", "a-real-secret-at-least-32-bytes-ok!!", "smtp", "", "postgres://x", "CORS_ORIGINS"},
		{"missing dsn", "production", "a-real-secret-at-least-32-bytes-ok!!", "smtp", "https://x.example", "", "DATABASE_URL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBoot(tc.env, tc.jwt, tc.mail, tc.cors, tc.dsn)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want substring %q", err, tc.want)
			}
		})
	}
}
