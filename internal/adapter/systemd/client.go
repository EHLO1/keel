package systemd

import (
	"context"
	"log/slog"
	"time"
)

type Client interface {
	NewClient(ctx context.Context, log *slog.Logger)
	Observe(ctx context.Context, svcs []string)
	Close()
}

type Service struct {
	Name        string
	ActiveState string
	SubState    string
}

type ServiceStatus struct {
	ObservedAt time.Time
	Services   []Service
}
