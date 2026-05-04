package services

import "github.com/EHLO1/keel/backend/internal/config"

type ValkeyClientService struct {
	cfg *config.Config
}

func NewValkeyClientService(cfg *config.Config) *ValkeyClientService {
	return &ValkeyClientService{
		cfg: cfg,
	}
}
