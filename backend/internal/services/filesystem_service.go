package services

import "github.com/EHLO1/keel/backend/internal/config"

type FilesystemService struct {
	cfg *config.Config
}

func NewFilesystemService(cfg *config.Config) *FilesystemService {
	return &FilesystemService{
		cfg: cfg,
	}
}
