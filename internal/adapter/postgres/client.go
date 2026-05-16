package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	postgres *pgxpool.Pool
	log      *slog.Logger
}

func NewClient(ctx context.Context, addr string, log *slog.Logger) *Client {
	poolConfig, _ := pgxpool.ParseConfig(addr)

	// Create a new PGX Pool using config.
	pool, _ := pgxpool.NewWithConfig(ctx, poolConfig)

	return &Client{
		postgres: pool,
		log:      log,
	}
}

func (c *Client) Observe(ctx context.Context) PostgresState {
	state := PostgresState{
		ObservedAt: time.Now(),
	}

	inRecovery, err := c.observeRole(ctx)
	if err != nil {
		c.log.Debug("postgres unreachable", "error", err)
		return state
	}

	state.Reachable = true

	if inRecovery {
		state.Role = string(RoleReplica)
		c.observeReplica(ctx, &state)
	} else {
		state.Role = string(RolePrimary)
		c.observePrimary(ctx, &state)
	}

	return state
}

func (c *Client) observeRole(ctx context.Context) (bool, error) {
	var inRecovery bool

	// If Postgres instance is in recovery, return true, otherwise return false (meaning Postgres is Primary)
	err := c.postgres.QueryRow(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery)
	if err != nil {
		return false, fmt.Errorf("unable to determine role: %w", err)
	}

	return !inRecovery, nil
}

func (c *Client) observeReplica(ctx context.Context, state *PostgresState) {
	const q = `
        SELECT
            pg_last_wal_receive_lsn()::text,
            pg_last_wal_replay_lsn()::text,
            COALESCE(
                (SELECT latest_end_lsn::text FROM pg_stat_wal_receiver),
                ''
            ),
            COALESCE(
                pg_wal_lsn_diff(
                    (SELECT latest_end_lsn FROM pg_stat_wal_receiver),
                    pg_last_wal_replay_lsn()
                ),
                -1
            ),
            EXISTS (SELECT 1 FROM pg_stat_wal_receiver)
    `

	var lagBytes int64
	err := c.postgres.QueryRow(ctx, q).Scan(
		&state.ReceiveLSN,
		&state.ReplayLSN,
		&state.UpstreamPrimaryLSN,
		&lagBytes,
		&state.StreamingActive,
	)
	if err != nil {
		c.log.Debug("observing standby state", "error", err)
		return
	}

	if lagBytes >= 0 {
		state.LagBytes = lagBytes
		state.LagKnown = true
	}
	// else: leave LagBytes=0 and LagKnown=false
}

func (c *Client) observePrimary(ctx context.Context, state *PostgresState) {
	if lsn, err := c.currentWriteLSN(ctx); err != nil {
		c.log.Debug("determining write lsn", "error", err)
	} else {
		state.CurrentWriteLSN = lsn
	}

	if replicas, err := c.listReplicas(ctx); err != nil {
		c.log.Debug("listing replicas", "error", err)
	} else {
		state.Replicas = replicas
	}
}

func (c *Client) currentWriteLSN(ctx context.Context) (string, error) {
	var lsn string
	err := c.postgres.QueryRow(ctx, "SELECT pg_current_wal_lsn()::text").Scan(&lsn)
	if err != nil {
		return "", err
	}
	return lsn, nil
}

func (c *Client) listReplicas(ctx context.Context) ([]Replica, error) {
	rows, err := c.postgres.Query(ctx, `
		SELECT
			application_name,
			state,
			COALESCE(sent_lsn):text, ''),
			COALESCE(write_lsn):text, ''),
			COALESCE(flush_lsn):text, ''),
			COALESCE(replay_lsn):text, ''),
			sync_state,
			pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS lag_bytes
		FROM pg_stat_replication
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying pg_stat_replication: %w", err)
	}
	defer rows.Close()

	var replicas []Replica
	for rows.Next() {
		var (
			r        Replica
			lagBytes sql.NullInt64
		)
		if err := rows.Scan(
			&r.ApplicationName,
			&r.State,
			&r.SentLSN,
			&r.WriteLSN,
			&r.FlushLSN,
			&r.ReplayLSN,
			&r.SyncState,
			&lagBytes,
		); err != nil {
			return nil, fmt.Errorf("error scanning replication row: %w", err)
		}
		if lagBytes.Valid {
			r.LagBytes = lagBytes.Int64
			r.LagKnown = true
		}
		replicas = append(replicas, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error processing replication rows: %w", err)
	}
	return replicas, nil
}
