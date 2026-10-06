package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string, timeout time.Duration) (*Database, error) {
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pool, err := pgxpool.New(connectCtx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return &Database{pool: pool}, nil
}

func (d *Database) Ping(ctx context.Context) error {
	if d == nil || d.pool == nil {
		return fmt.Errorf("database is not initialised")
	}

	if err := d.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return nil
}

func (d *Database) Exec(ctx context.Context, sql string) error {
	if d == nil || d.pool == nil {
		return fmt.Errorf("database is not initialised")
	}

	if _, err := d.pool.Exec(ctx, sql); err != nil {
		return fmt.Errorf("execute PostgreSQL statement: %w", err)
	}

	return nil
}

func (d *Database) Close() {
	if d != nil && d.pool != nil {
		d.pool.Close()
	}
}
