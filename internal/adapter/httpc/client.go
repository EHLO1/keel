package httpc

import (
	"net/http"
	"time"
)

type Client struct {
	httpC *http.Client
}

func NewClient() *Client {
	return &Client{
		httpC: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
