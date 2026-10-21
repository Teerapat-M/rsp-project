package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

const leakedDSNPassword = "leaked-password-xyz"

func TestConnectPostgresRedactsPasswordOnFailure(t *testing.T) {
	// Closed port + parse-unfriendly DSN both go through pgx. Either path
	// must never print the password (the live Render bug: migrate/pgx embed
	// the full DSN in "failed to open database, \"...\"").
	cases := []string{
		"postgres://leakuser:" + leakedDSNPassword + "@127.0.0.1:1/postgres?sslmode=disable",
		"postgres://leakuser:" + leakedDSNPassword + "@%zz",
		"postgres://leakuser:" + leakedDSNPassword + "@",
		`postgres://leakuser:` + leakedDSNPassword + `@db.invalid:5432/postgres?sslmode=require`,
	}
	for i, dsn := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			pool, err := connectPostgres(ctx, dsn)
			if pool != nil {
				pool.Close()
			}
			if err == nil {
				t.Fatal("expected a connection error")
			}
			msg := err.Error()
			if strings.Contains(msg, leakedDSNPassword) {
				t.Fatalf("password leaked in %q", msg)
			}
			if strings.Contains(msg, "leakuser") {
				t.Fatalf("userinfo leaked in %q", msg)
			}
		})
	}
}
