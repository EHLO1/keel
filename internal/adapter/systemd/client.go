package systemd

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
)

// Client wraps the systemd D-Bus connection.
type Client struct {
	conn *dbus.Conn
}

// New establishes a connection to the system bus.
func NewClient(ctx context.Context) (*Client, error) {
	// Connect to the system bus (requires root/sudo when running the daemon)
	conn, err := dbus.NewSystemdConnectionContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd dbus: %w", err)
	}

	return &Client{
		conn: conn,
	}, nil
}

// Close gracefully closes the D-Bus connection. Call this during shutdown.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Status represents the current status of a systemd service.
type Status struct {
	ActiveState string // e.g., "active", "inactive", "failed"
	SubState    string // e.g., "running", "exited", "dead"
}

// GetState fetches the exact current state of a unit (e.g., "keepalived.service").
func (c *Client) GetStatus(ctx context.Context, svcName string) (*Status, error) {
	// Fetch ActiveState
	activeProp, err := c.conn.GetUnitPropertyContext(ctx, svcName, "ActiveState")
	if err != nil {
		return nil, fmt.Errorf("failed to get ActiveState for %s: %w", svcName, err)
	}

	// Fetch SubState (gives more granularity, like if it's active but "exited")
	subProp, err := c.conn.GetUnitPropertyContext(ctx, svcName, "SubState")
	if err != nil {
		return nil, fmt.Errorf("failed to get SubState for %s: %w", svcName, err)
	}

	// dbus.Property Values are wrapped interfaces, we must assert them to strings
	activeState, ok1 := activeProp.Value.Value().(string)
	subState, ok2 := subProp.Value.Value().(string)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("failed to parse systemd properties for %s as strings", svcName)
	}

	return &Status{
		ActiveState: activeState,
		SubState:    subState,
	}, nil
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
