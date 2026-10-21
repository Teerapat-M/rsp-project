package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Teerapat-M/rsp-project/internal/hub"
	"github.com/Teerapat-M/rsp-project/internal/rooms"
	"github.com/Teerapat-M/rsp-project/internal/server"
	"github.com/Teerapat-M/rsp-project/internal/sqlcdb"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	addr := getenv("HTTP_ADDR", ":8080")
	dsn := getenv("DATABASE_URL", "postgres://rsp:rsp@localhost:5432/rsp?sslmode=disable")
	grace := 20 * time.Second
	if v := os.Getenv("DISCONNECT_GRACE"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("DISCONNECT_GRACE: %v", err)
		}
		grace = d
	}

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
	h := hub.New(store, hub.Config{Grace: grace})
	r := server.New(h).Router()

	log.Printf("listening on %s (phase 2 rooms/ws, grace=%s)", addr, grace)
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
