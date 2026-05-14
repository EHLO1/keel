package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (c *Client) IsPrimary(ctx context.Context) (*PostgresState, error) {
	p := &PostgresState{Reachable: false}
	var inRecovery bool

	// If Postgres instance is in recovery, return true, otherwise return false (meaning Postgres is Primary)
	err := c.postgres.QueryRow(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery)
	switch {
	case err != nil:
		return
	}

	p.Reachable = true
	return !inRecovery, nil

}

func (c *Client) WALStateWithDiff(ctx context.Context, replicaAppName string) (*PostgresState, error) {

	query := `
		SELECT
			pg_current_wal_lsn()::text,
			replay_lsn::text,
			pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn)
		FROM pg_stat_replication
		WHERE application_name = $1;
	`
	p := &PostgresState{Reachable: false}

	err := c.postgres.QueryRow(ctx, query, replicaAppName).Scan(
		&p.CurrentWriteLSN,
		&p.ReplayLSN,
		&p.LagBytes,
	)

	if err != nil {
		// CRITICAL ORCHESTRATOR LOGIC:
		// If a replica's network drops, it disappears from pg_stat_replication entirely.
		// QueryRow will return sql.ErrNoRows. Handle this specifically so the
		// orchestrator knows the replica is dead, rather than just throwing a generic DB error.
		if errors.Is(err, pgx.ErrNoRows) {
			return &PostgresState{
				Reachable:       true,
				StreamingActive: false,
			}, nil
		}

		// Remaining standard connection and query errors.
		return nil, fmt.Errorf("failed to query replication state: %w", err)
	}

	p.Reachable = true
	return p, nil
}

func (c *Client) CurrentWriteLSN(ctx context.Context) (*PostgresState, error) {
	p := &PostgresState{Reachable: false}

	err := c.postgres.QueryRow(ctx, "SELECT pg_current_wal_lsn()::text").Scan(&p.CurrentWriteLSN)
	if err != nil {
		return nil, err
	}

	p.Reachable = true
	return p, nil
}

func (c *Client) ReplayLSNWithDiff(ctx context.Context, targetLSN string) (*PostgresState, error) {
	p := &PostgresState{Reachable: false}

	query := `
		SELECT
			pg_last_wal_replay_lsn()::text,
			pg_wal_lsn_diff(pg_last_wal_replay_lsn(), $1::pg_lsn)
	`

	// QueryRow executes the query and Scan maps the two selected columns to our two variables.
	err := c.postgres.QueryRow(ctx, query, targetLSN).Scan(&p.ReplayLSN, &p.LagBytes)
	if err != nil {
		return nil, err
	}

	p.Reachable = true
	return p, nil
}

func (c *Client) Reachable(ctx context.Context) bool {
	err := c.postgres.Ping(ctx)
	return err == nil
}
