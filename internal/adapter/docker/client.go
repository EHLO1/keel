package docker

import (
	"context"
	"fmt"

	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/volume"
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
