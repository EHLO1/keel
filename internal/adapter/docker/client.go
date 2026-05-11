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
	dockerClient, err := client.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %s", err)
	}
	return &Client{docker: dockerClient}, nil
}
