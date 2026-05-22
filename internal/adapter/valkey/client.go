package valkey

import (
	"bufio"
	"context"
	"fmt"
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

	switch role {
	case "master":
		c.observeCommon(&state, infoData)
	case "replica", "slave":
		c.observeCommon(&state, infoData)
		c.observeReplica(&state, infoData)
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

func (c *Client) observeCommon(state *ValkeyState, info InfoMap) {
	for key, valueStr := range info {

		// Handle list of connected replicas
		isReplicaField := strings.HasPrefix(key, "slave") || strings.HasPrefix(key, "replica")
		if isReplicaField && strings.Contains(valueStr, "=") {
			replica := c.parseReplicaString(valueStr)
			state.Replicas = append(state.Replicas, replica)
			continue
		}

		switch key {

		case "connected_slaves", "connected_replicas":
			if val, err := strconv.Atoi(valueStr); err == nil {
				state.ConnectedReplicas = val
			}

		case "replicas_waiting_psync":
			if val, err := strconv.Atoi(valueStr); err == nil {
				state.ReplicasWaitingPsync = val
			}

		case "master_failover_state", "primary_failover_state":
			switch valueStr {
			case "waiting-for-sync":
				state.PrimaryFailoverState = WaitingForSync
			case "failover-in-progress":
				state.PrimaryFailoverState = InProgress
			default:
				state.PrimaryFailoverState = None
			}

		case "master_replid", "primary_replid":
			state.PrimaryReplId = valueStr

		case "master_replid2", "primary_replid2":
			state.SecondReplId = valueStr

		case "master_repl_offset", "primary_repl_offset":
			if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
				state.PrimaryReplOffset = val
			}

		case "second_repl_offset":
			if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
				state.SecondReplOffset = val
			}

		case "repl_backlog_active":
			if val, err := strconv.ParseBool(valueStr); err == nil {
				state.ReplBacklogActive = val
			}

		case "repl_backlog_size":
			if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
				state.ReplBacklogSize = val
			}

		// No matches
		default:
			continue
		}
	}
}

func (c *Client) parseReplicaString(valStr string) Replica {
	var r Replica

	fields := strings.Split(valStr, ",")

	for _, field := range fields {
		kv := strings.SplitN(field, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key, val := kv[0], kv[1]

		switch key {
		case "ip":
			r.IP = val
		case "port":
			if p, err := strconv.Atoi(val); err == nil {
				r.Port = p
			}
		case "state":
			r.State = val
		case "offset":
			if o, err := strconv.ParseInt(val, 10, 64); err == nil {
				r.Offset = o
			}
		case "lag":
			if l, err := strconv.Atoi(val); err == nil {
				r.Lag = l
			}
		}
	}

	return r
}

func (c *Client) observeReplica(state *ValkeyState, info InfoMap) {
	for key, valueStr := range info {

		switch key {

		case "master_link_status", "primary_link_status":
			state.PrimaryLinkStatus = valueStr

		case "master_last_io_seconds_ago", "primary_last_io_seconds_ago":
			if val, err := strconv.Atoi(valueStr); err == nil {
				state.PrimaryLastIOSecondsAgo = val
			}

		case "master_sync_in_progress", "primary_sync_in_progress":
			if val, err := strconv.ParseBool(valueStr); err == nil {
				state.PrimarySyncInProgress = val
			}

		case "slave_read_repl_offset", "replica_read_repl_offset", "replica_read_offset":
			if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
				state.ReplicaReadReplOffset = val
			}

		case "slave_repl_offset", "replica_repl_offset", "replica_offset":
			if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
				state.ReplicaReplOffset = val
			}

		case "replicas_repl_buffer_size":
			if val, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
				state.ReplicaReplBufferSize = val
			}

		case "slave_priority", "replica_priority":
			if val, err := strconv.Atoi(valueStr); err == nil {
				state.ReplicaPriority = val
			}

		case "slave_read_only", "replica_read_only":
			if val, err := strconv.ParseBool(valueStr); err == nil {
				state.ReplicaReadOnly = val
			}

		// No matches
		default:
			continue
		}
	}
}

func (c *Client) PromoteToMaster(ctx context.Context) error {
	c.log.Info("promoting valkey to master")
	err := c.valkey.Do(ctx, "REPLICAOF", "NO", "ONE").Err()
	if err != nil {
		return fmt.Errorf("failed to promote valkey to master: %w", err)
	}
	return nil
}

func (c *Client) MakeReplicaOf(ctx context.Context, host string, port int) error {
	c.log.Info("setting valkey replica of", "host", host, "port", port)
	err := c.valkey.Do(ctx, "REPLICAOF", host, strconv.Itoa(port)).Err()
	if err != nil {
		return fmt.Errorf("failed to configure valkey replication: %w", err)
	}
	return nil
}
