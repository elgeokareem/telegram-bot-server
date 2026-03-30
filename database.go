package main

import (
	"bottelegram/server/config"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreatePool(env config.Env) (*pgxpool.Pool, error) {
	dbURL := fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=disable", env.DBSchema, env.DBUser, env.DBPassword, env.DBHost, env.DBPort, env.DBName)

	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	poolCfg.MaxConns = 10
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect to db: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}
