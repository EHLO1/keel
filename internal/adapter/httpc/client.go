package httpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	httpC        *http.Client
	stateAPIPort string
	log          *slog.Logger
}

func NewClient(stateAPIPort string, log *slog.Logger) *Client {
	return &Client{
		httpC: &http.Client{
			Timeout: 5 * time.Second,
		},
		stateAPIPort: stateAPIPort,
		log:          log,
	}
}

func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpC.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) ObservePeerKeelStateAPI(ctx context.Context, peerAddr string) (PeerState, bool) {
	var st PeerState

	host := net.JoinHostPort(peerAddr, c.stateAPIPort)

	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/api/state",
	}

	queryCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	respBytes, err := c.Get(queryCtx, u.String())
	if err != nil {
		c.log.Debug("failed to GET peer state", "error", err)
		return st, false
	}

	if err := json.Unmarshal(respBytes, &st); err != nil {
		c.log.Debug("failed to unmarshal state", "error", err)
		return st, true
	}

	return st, true
}
