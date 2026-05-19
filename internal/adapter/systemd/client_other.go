//go:build !linux

package systemd

import (
	"context"
	"log/slog"
	"time"
)

type mockClient struct {
	log *slog.Logger
}

func NewClient(ctx context.Context, log *slog.Logger) (Client, error) {
	log.Warn("Systemd dbus not supported on this OS. Using mock client.")
	return &mockClient{log: log}, nil
}

func (c *mockClient) Observe(ctx context.Context, svcs []string) ServiceStatus {
	result := ServiceStatus{
		ObservedAt: time.Now(),
		Services:   make([]Service, len(svcs)),
	}

	for i, svc := range svcs {
		result.Services[i] = Service{
			Name:        svc,
			ActiveState: "unknown",
			SubState:    "os-not-supported",
		}
	}

	return result
}

func (c *mockClient) Close() {
	// No-op
}
