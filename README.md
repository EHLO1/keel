# Keel Orchestrator
Orchestrator for 2+ node backends using WireGuard and Keepalived. Supports Postgres and Redis/Valkey.

## Requirements
- Linux
- WireGuard
- Keepalived
- Repmgr Plugin (Postgres)

## Function
1. Reconciler acts as timer and trigger.
2. Reconciler calls State to collect a snapshot, containing the results of multiple checks.
3. State calls Services to build a snapshot.
4. Services collect telemetry, such as WireGuard Tunnel health, Postgres role, Valkey role, and more.
5. State delivers snapshot to Reconciler.
6. Reconciler calls Policy to evaluate the snapshot and determine next steps.
7. Policy evaluates the snapshot and creates a desired state containing any declarative changes to be made.
8. Policy delivers desired state to Reconciler.
9. Reconciler calls Actor to make the current state match the desired state.
10. Actor calls Services to apply desired state.

## Overview
Snapshot data (Environment State) is kept in memory only, by design. When a node goes offline, it should always come online in an unhealthy/backup state. The orchestrator will then build a new snapshot from scratch. Nothing is assumed. We want to ensure that primary roles are never preempted (retaken) prematurely. Keel's primary jobs are to:
- Inform Keepalived of the overall "healthy / unheathy" state of the node, thereby enabling Keepalived to preempt MASTER or BACKUP accordingly
- Execute actual failover steps in Postgres and Valkey/Redis.