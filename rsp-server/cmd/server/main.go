package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/auth"
	"github.com/Teerapat-M/rsp-project/internal/hub"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"github.com/Teerapat-M/rsp-project/internal/server"
	"github.com/Teerapat-M/rsp-project/internal/sqlcdb"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	addr := getenv("HTTP_ADDR", ":8080")
	appEnv := getenv("APP_ENV", "dev")
	jwtSecret := os.Getenv("JWT_SECRET")
	emailProvider := getenv("EMAIL_PROVIDER", "log")
	corsCSV := os.Getenv("CORS_ORIGINS")
	dsnEnv := os.Getenv("DATABASE_URL")
	if err := validateBoot(appEnv, jwtSecret, emailProvider, corsCSV, dsnEnv); err != nil {
		log.Fatal(err)
	}

	dsn := dsnEnv
	if dsn == "" {
		dsn = "postgres://rsp:rsp@localhost:5433/rsp?sslmode=disable"
	}

	grace := 20 * time.Second
	if v := os.Getenv("DISCONNECT_GRACE"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("DISCONNECT_GRACE: %v", err)
		}
		grace = d
	}

	secret := []byte(jwtSecret)
	if len(secret) == 0 {
		log.Print("JWT_SECRET unset; using a dev-only default")
		secret = []byte(auth.DevJWTSecret)
	}

	origins := server.ParseOrigins(corsCSV)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("postgres ping: %v", err)
	}

	store := rooms.NewPostgres(sqlcdb.New(pool))
	h := hub.New(store, hub.Config{
		Grace: grace,
		CheckOrigin: func(r *http.Request) bool {
			return server.OriginAllowed(r.Header.Get("Origin"), origins)
		},
	})

	var google auth.GoogleVerifier
	if cid := os.Getenv("GOOGLE_CLIENT_ID"); cid != "" {
		google = auth.TokenInfoVerifier{ClientID: cid}
	} else {
		log.Print("GOOGLE_CLIENT_ID unset; Google sign-in disabled")
	}

	mail := auth.SenderFromEnv(
		emailProvider,
		os.Getenv("RESEND_API_KEY"),
		os.Getenv("EMAIL_FROM"),
		os.Getenv("SMTP_HOST"),
		getenv("SMTP_PORT", "587"),
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASSWORD"),
	)

	cost := getenvInt("BCRYPT_COST", 12)
	if cost < 10 {
		cost = bcrypt.DefaultCost
	}

	a := auth.New(store, auth.Config{
		Secret:          secret,
		AccessTTL:       getenvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTTL:      getenvDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		RefreshRemember: getenvDuration("REFRESH_TOKEN_TTL_REMEMBER", 30*24*time.Hour),
		ResetTTL:        getenvDuration("PASSWORD_RESET_TTL", time.Hour),
		BcryptCost:      cost,
		AppEnv:          appEnv,
		PublicBase:      getenv("PUBLIC_BASE_URL", "http://localhost:5173"),
		Mail:            mail,
		Google:          google,
	})

	limit := getenvInt("RATE_LIMIT_AUTH_PER_MIN", 20)
	if limit < 1 {
		limit = 20
	}
	api := server.New(h, a)
	api.CORSOrigins = origins
	api.AuthLimit = limit
	api.Ping = pool.Ping
	r := api.Router()

	log.Printf("listening on %s (phase 6 deploy, env=%s, grace=%s)", addr, appEnv, grace)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Fatalf("%s: %v", key, err)
	}
	return d
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("%s: %v", key, err)
	}
	return n
}
