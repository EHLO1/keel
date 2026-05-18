//go:build linux

package systemd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

type systemdClient struct {
	conn *dbus.Conn
	log  *slog.Logger
}

func NewClient(ctx context.Context, log *slog.Logger) (*systemdClient, error) {
	// Connect to the system bus (requires root/sudo)
	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd dbus: %w", err)
	}

	return &systemdClient{
		conn: conn,
		log:  log,
	}, nil
}

func (c *systemdClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *systemdClient) Observe(ctx context.Context, svcs []string) *ServiceStatus {
	result := &ServiceStatus{
		ObservedAt: time.Now(),
		Services:   make([]Service, len(svcs)),
	}

	// Batch status request
	units, err := c.conn.ListUnitsByNamesContext(ctx, svcs)
	if err != nil {
		c.log.Error("failed to list systemd services", "error", err)
		for i, svc := range svcs {
			result.Services[i] = Service{
				Name:        svc,
				ActiveState: "unknown",
				SubState:    "dbus-error",
			}
		}
		return result
	}

	unitMap := make(map[string]dbus.UnitStatus, len(units))
	for _, u := range units {
		unitMap[u.Name] = u
	}

	for i, svc := range svcs {
		if u, ok := unitMap[svc]; ok {
			result.Services[i] = Service{
				Name:        u.Name,
				ActiveState: u.ActiveState,
				SubState:    u.SubState,
			}
		} else {
			result.Services[i] = Service{
				Name:        svc,
				ActiveState: "unknown",
				SubState:    "dbus-error",
			}
			c.log.Debug("service missing from dbus response", "service", svc)
		}

	}

	return result
}

// // StartUnit safely starts a service and waits for it to complete.
// func (c *Client) StartService(ctx context.Context, svcName string) error {
// 	return c.executeJob(ctx, svcName, c.conn.StartUnitContext)
// }

// // StopUnit safely stops a service and waits for it to complete.
// func (c *Client) StopService(ctx context.Context, svcName string) error {
// 	return c.executeJob(ctx, svcName, c.conn.StopUnitContext)
// }

// // RestartUnit safely restarts a service and waits for it to complete.
// func (c *Client) RestartService(ctx context.Context, svcName string) error {
// 	return c.executeJob(ctx, svcName, c.conn.RestartUnitContext)
// }

// // executeJob is an internal helper that handles the asynchronous nature of systemd.
// func (c *Client) executeJob(
// 	ctx context.Context,
// 	svcName string,
// 	actionFunc func(context.Context, string, string, chan<- string) (int, error),
// ) error {
// 	// Systemd sends the result of the job to this channel
// 	ch := make(chan string)

// 	// "replace" mode tells systemd: if there are other pending jobs for this unit,
// 	// cancel them and do this one instead. This prevents job queue gridlock.
// 	_, err := actionFunc(ctx, svcName, "replace", ch)
// 	if err != nil {
// 		return fmt.Errorf("failed to queue systemd job for %s: %w", svcName, err)
// 	}

// 	// Wait for the job to finish, or for our context to timeout/cancel
// 	select {
// 	case <-ctx.Done():
// 		return fmt.Errorf("context canceled while waiting for systemd job on %s: %w", svcName, ctx.Err())
// 	case result := <-ch:
// 		if result != "done" {
// 			// Results can be "done", "canceled", "timeout", "failed", "dependency", or "skipped"
// 			return fmt.Errorf("systemd job for %s did not succeed, result: %s", svcName, result)
// 		}
// 		return nil
// 	}
// }
