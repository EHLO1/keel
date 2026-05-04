package services

import "github.com/EHLO1/keel/backend/internal/config"

type PostgresProbeService struct {
	cfg *config.Config
}

func NewPostgresProbeService(cfg *config.Config) *PostgresProbeService {
	return &PostgresProbeService{
		cfg: cfg,
	}
}
