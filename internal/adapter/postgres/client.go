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

func (s *Client) IsPrimary(ctx context.Context) (bool, error) {
	var inRecovery bool

	// If Postgres instance is in recovery, return true, otherwise return false (meaning Postgres is Primary)
	if err := s.postgres.QueryRow(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery); err != nil {
		return false, err
	}

	return !inRecovery, nil

}
