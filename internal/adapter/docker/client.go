package docker

import (
	"context"
	"fmt"

	"github.com/docker/go-sdk/client"
)

type Client struct {
	docker client.SDKClient
}

func NewClient(ctx context.Context) (*Client, error) {
	cli, err := client.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %s", err)
	}
	return &Client{docker: cli}, nil
}

func (c *Client) Close() error {
	return c.docker.Close()
}

func (c *Client) GetVolumeMountpoint(ctx context.Context, volumeName string) (string, error) {
	vol, err := c.docker.VolumeList()
}
