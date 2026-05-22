package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/volume"
	mobyclient "github.com/moby/moby/client"
)

type Client struct {
	docker    client.SDKClient
	pgVolName string
}

func NewClient(ctx context.Context, pgVolName string) (*Client, error) {
	cli, err := client.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %s", err)
	}
	return &Client{
		docker:    cli,
		pgVolName: pgVolName,
	}, nil
}

func (c *Client) Close() error {
	return c.docker.Close()
}

func (c *Client) GetVolumeMountpoint(ctx context.Context) (string, error) {
	vol, err := volume.FindByID(ctx, c.pgVolName)
	if err != nil {
		return "", fmt.Errorf("docker volume does not exist: %s", err)
	}
	return vol.Mountpoint, nil
}

func (c *Client) RestartPostgresContainer(ctx context.Context) error {
	res, err := c.docker.ContainerList(ctx, mobyclient.ContainerListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list docker containers: %w", err)
	}

	var targetID string
	for _, item := range res.Items {
		matchesVolume := false
		for _, m := range item.Mounts {
			if m.Name == c.pgVolName {
				matchesVolume = true
				break
			}
		}

		if matchesVolume {
			targetID = item.ID
			break
		}
	}

	if targetID == "" {
		for _, item := range res.Items {
			for _, name := range item.Names {
				if strings.Contains(strings.ToLower(name), "postgres") {
					targetID = item.ID
					break
				}
			}
			if targetID != "" {
				break
			}
		}
	}

	if targetID == "" {
		return fmt.Errorf("could not find container mounting volume %s or named postgres", c.pgVolName)
	}

	_, err = c.docker.ContainerRestart(ctx, targetID, mobyclient.ContainerRestartOptions{})
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w", targetID, err)
	}
	return nil
}
