package services

import "github.com/EHLO1/keel/backend/internal/config"

type PostgresClientService struct {
	cfg *config.Config
}

func NewPostgresClientService(cfg *config.Config) *PostgresClientService {
	return &PostgresClientService{
		cfg: cfg,
	}
}
