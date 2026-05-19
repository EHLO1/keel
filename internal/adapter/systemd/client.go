package systemd

import (
	"context"
	"time"
)

type Client interface {
	Observe(ctx context.Context, svcs []string) ServiceStatus
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
