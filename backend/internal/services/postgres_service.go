package services

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresService struct {
	pool *pgxpool.Pool
}

func NewPostgresClientService(pool *pgxpool.Pool) *PostgresService {
	return &PostgresService{
		pool: pool,
	}
}

func (s *PostgresService) IsPrimary(ctx context.Context) (bool, error) {
	var inRecovery bool

	// If Postgres instance is in recovery, return true, otherwise return false (meaning Postgres is Primary)
	if err := s.pool.QueryRow(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery); err != nil {
		return false, err
	}

	return !inRecovery, nil

}

func (s *PostgresService) Promote
