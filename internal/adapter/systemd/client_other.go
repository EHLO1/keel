//go:build !linux

package systemd

import (
	"context"
	"fmt"
	"log/slog"
)

type mockClient struct {
}

func NewClient(ctx context.Context, log *slog.Logger) (Client, error) {
	return &mockClient{}, nil
}

func (c *mockClient) Status() error {
	return fmt.Errorf("Systemd Service Manager not supported on this OS")
}
