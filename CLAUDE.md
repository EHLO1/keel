# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## How I want to work with you

The user is learning Go and building this project to understand it deeply. **Default behavior: do not write code unless explicitly asked.** Instead:
- Explain concepts and walk through reasoning
- Ask Socratic questions to guide understanding
- When the user shares code, identify what could improve and explain why — don't rewrite it
- When proposing solutions, lay out tradeoffs rather than handing over an answer
- Reference real Go idioms and point to standard library or well-known project examples rather than novel implementations

## Build & Test Commands

```bash
# Build main binary
go build -o keel ./cmd/keel

# Build backend binary
go build -o keel-backend ./backend/cmd

# Run all tests
go test ./...

# Run tests in a specific package (verbose)
go test -v ./internal/config/...
go test -v ./backend/internal/services/...

# Run a single test by name
go test -v -run TestName ./path/to/package

# Run the app
./keel run [--env /path/to/.env]
```

## Architecture

Keel orchestrates two-node high-availability clusters (PostgreSQL primary/replica + Valkey) over a WireGuard tunnel, managed by Keepalived/VRRP. It is **Linux-only** and depends on WireGuard kernel module, systemd, and Keepalived being present.

### Reconciliation Loop

Every tick interval the system runs this cycle:

1. **Reconciler** (`backend/internal/services/reconciler_service.go`) — timer that drives each cycle
2. **StateService** (`backend/internal/services/state_service.go`) — fans out concurrent probes and returns an atomic `Snapshot`
3. **PolicyService** (`backend/internal/services/policy_service.go`) — inspects the snapshot and determines desired state changes
4. **ActorService** (`backend/internal/services/actor_service.go`) — applies desired state to real systems (Keepalived, Postgres, Valkey, etc.)

All state is in-memory by design; nodes start unhealthy and rebuild from scratch on each run.

### Module Structure

There are **two Go modules**:
- Root (`go.mod`) — hosts the HTTP API server (`internal/api/`) and thin adapters (`internal/adapter/`) wrapping external systems (postgres, valkey, wireguard, systemd, http, icmp, network, filesystem)
- `backend/` (`backend/go.mod`) — hosts the core orchestration services (StateService, ReconcilerService, PolicyService, ActorService, and per-system client services)

### Configuration

`internal/config/config.go` and `backend/internal/config/config.go` use reflection to load config from environment variables. Key conventions:
- Struct field tags: `env:"VAR_NAME"`, `default:"value"`, `options:"file,toLower,trimTrailingSlash"`
- Docker secrets pattern: `{VAR}_FILE` suffix loads the value from a file
- Source precedence: real env vars > `.env` file > struct tag defaults
- The `--env` CLI flag overrides the `.env` file path

### Snapshot & Atomics

`internal/types/snapshot.go` defines the `Snapshot` struct. The running snapshot is stored as `atomic.Pointer[Snapshot]` for lock-free reads across goroutines.

### Adapters vs Services

- **Adapters** (`internal/adapter/`) — thin wrappers around external systems with no business logic
- **Services** (`backend/internal/services/`) — business logic; compose adapters and make decisions

### Anti-flap / Hysteresis

`PeerDownStrikes` in the snapshot tracks consecutive missed probes to avoid premature role transitions. Primary roles are never preempted prematurely.
