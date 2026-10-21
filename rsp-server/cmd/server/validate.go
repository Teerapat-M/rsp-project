package main

import (
	"fmt"
	"strings"

	"github.com/Teerapat-M/rsp-project/internal/auth"
	"github.com/Teerapat-M/rsp-project/internal/server"
)

func isProduction(env string) bool {
	e := strings.ToLower(strings.TrimSpace(env))
	return e == "production" || e == "prod"
}

func validateBoot(appEnv, jwtSecret, emailProvider, corsOrigins, databaseURL string) error {
	if !isProduction(appEnv) {
		return nil
	}
	if strings.TrimSpace(jwtSecret) == "" {
		return fmt.Errorf("JWT_SECRET is required when APP_ENV=production")
	}
	if jwtSecret == auth.DevJWTSecret {
		return fmt.Errorf("JWT_SECRET must not be the documented dev default in production")
	}
	p := strings.ToLower(strings.TrimSpace(emailProvider))
	if p == "" || p == "log" {
		return fmt.Errorf("EMAIL_PROVIDER=log is forbidden when APP_ENV=production (use resend or smtp)")
	}
	if p != "resend" && p != "smtp" {
		return fmt.Errorf("EMAIL_PROVIDER must be resend or smtp when APP_ENV=production")
	}
	if len(server.ParseOrigins(corsOrigins)) == 0 {
		return fmt.Errorf("CORS_ORIGINS is required when APP_ENV=production")
	}
	if strings.TrimSpace(databaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required when APP_ENV=production")
	}
	return nil
}
