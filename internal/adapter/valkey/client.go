package valkey

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	valkey *redis.Client
}

func NewClient(ctx context.Context, addr string, password string, db int) *Client {
	conn := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &Client{valkey: conn}
}
