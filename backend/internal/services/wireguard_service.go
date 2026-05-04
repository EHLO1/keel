package services

import (
	"github.com/EHLO1/keel/backend/internal/config"
)

type WireguardService struct {
	cfg *config.Config
}

func NewWireguardService(cfg *config.Config) *WireguardService {
	return &WireguardService{
		cfg: cfg,
	}
}
