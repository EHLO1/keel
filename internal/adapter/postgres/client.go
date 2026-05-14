package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	postgres *pgxpool.Pool
}

func NewClient(ctx context.Context, addr string) *Client {
	poolConfig, _ := pgxpool.ParseConfig(addr)

	// Create a new PGX Pool using config.
	pool, _ := pgxpool.NewWithConfig(ctx, poolConfig)

	return &Client{
		postgres: pool,
	}
}

func (c *Client) IsPrimary(ctx context.Context) (bool, error) {
	var inRecovery bool

	// If Postgres instance is in recovery, return true, otherwise return false (meaning Postgres is Primary)
	if err := c.postgres.QueryRow(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery); err != nil {
		return false, err
	}

	return !inRecovery, nil

}

func (c *Client) GetPrimaryLSN(ctx context.Context) (string, error) {
	var lsn string
	if err := c.postgres.QueryRow(ctx, "SELECT pg_current_wal_lsn()").Scan(&lsn); err != nil {
		return "", err
	}
	return lsn, nil
}
