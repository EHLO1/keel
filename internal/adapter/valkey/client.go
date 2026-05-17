package valkey

import (
	"bufio"
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	valkey *redis.Client
	log    *slog.Logger
}

func NewClient(ctx context.Context, addr string, password string, db int, log *slog.Logger) *Client {
	conn := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &Client{
		valkey: conn,
		log:    log,
	}
}

func (c *Client) Observe(ctx context.Context) ValkeyState {
	state := ValkeyState{
		ObservedAt: time.Now(),
	}

	infoData := c.parseInfo(ctx)

	state.Reachable = true
	role, _ := infoData["role"]
	state.Role = role

	// Common Fields
	if offsetStr, ok := infoData["master_repl_offset"]; ok {
		if val, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			state.MasterReplOffset = val
		}
	}

	// Role-Specific Fields
	switch role {
	case "master":
		observeMaster(state, infoData)
	case "replica", "slave":
		observeReplica(state, infoData)
	}

	return state
}

func (c *Client) parseInfo(ctx context.Context) InfoMap {
	infoRaw, err := c.valkey.Info(ctx, "replication").Result()
	if err != nil {
		c.log.Debug("info command failed", "error", err)
		return nil
	}
	infoParsed := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(infoRaw))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			infoParsed[parts[0]] = parts[1]
		}
	}

	return infoParsed
}

func (c *Client) observeMaster(state ValkeyState, info InfoMap) {

}

func (c *Client) observeReplica(state ValkeyState, info InfoMap) {

}
