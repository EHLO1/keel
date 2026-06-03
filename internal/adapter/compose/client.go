package compose

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/EHLO1/keel/internal/adapter/proc"
)

type Runner interface {
	Run(ctx context.Context, dir string, env []string, name string, args ...string) (proc.Result, error)
}

type Client struct {
	runner     Runner
	dockerPath string
	env        []string
	log        *slog.Logger
}

func NewClient(runner Runner, env []string, log *slog.Logger) *Client {
	return &Client{
		runner: runner,
		env:    env,
		log:    log,
	}
}

func (c *Client) StackUp(ctx context.Context, stackName string) error {
	switch stackName {
	case "frontend", "app":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "app-compose.yaml", "up", "-d")
		return nil
	case "backend", "db":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "db-compose.yaml", "up", "-d")
		return nil
	default:
		return fmt.Errorf("stackName must be frontend, app, backend, or db")
	}
}

func (c *Client) StackDown(ctx context.Context, stackName string) error {
	switch stackName {
	case "frontend", "app":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "app-compose.yaml", "down")
		return nil
	case "backend", "db":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "db-compose.yaml", "down")
		return nil
	default:
		return fmt.Errorf("stackName must be frontend, app, backend, or db")
	}
}

func (c *Client) ServiceUp(ctx context.Context, stackName string, serviceName string) error {
	switch stackName {
	case "frontend", "app":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "app-compose.yaml", serviceName, "up", "-d")
		return nil
	case "backend", "db":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "db-compose.yaml", serviceName, "up", "-d")
		return nil
	default:
		return fmt.Errorf("stackName must be frontend, app, backend, or db")
	}
}

func (c *Client) ServiceDown(ctx context.Context, stackName string, serviceName string) error {
	switch stackName {
	case "frontend", "app":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "app-compose.yaml", serviceName, "down", "-d")
		return nil
	case "backend", "db":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "db-compose.yaml", serviceName, "down", "-d")
		return nil
	default:
		return fmt.Errorf("stackName must be frontend, app, backend, or db")
	}
}

func (c *Client) ServiceRestart(ctx context.Context, stackName string, serviceName string) error {
	switch stackName {
	case "frontend", "app":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "app-compose.yaml", serviceName, "restart")
		return nil
	case "backend", "db":
		c.runner.Run(ctx, c.dockerPath, c.env, "docker", "compose", "-f", "db-compose.yaml", serviceName, "restart")
		return nil
	default:
		return fmt.Errorf("stackName must be frontend, app, backend, or db")
	}
}
