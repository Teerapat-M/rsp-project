package main

import (
	"context"
	"fmt"

	"github.com/Teerapat-M/rsp-project/internal/redact"
	"github.com/jackc/pgx/v5/pgxpool"
)

func connectPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: %s", redact.DSNError(dsn, err))
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %s", redact.DSNError(dsn, err))
	}
	return pool, nil
}
