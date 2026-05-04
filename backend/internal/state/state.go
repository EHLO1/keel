// internal/state/state.go
package state

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/EHLO1/keel/backend/internal/config"
	"github.com/EHLO1/keel/backend/internal/probe"
)

// Snapshot is the immutable point-in-time view of the world.
type Snapshot struct {
	Vrrp              VrrpState
	LocalPg           PgRole
	LocalValkey       ValkeyRole
	ValkeyMasterHost  string
	PeerWgReachable   bool
	PeerPhysReachable bool
	PeerPgRole        PgRole
	PeerValkeyRole    ValkeyRole
	WgHandshakeAge    time.Duration
	LBReachable       bool
	Maintenance       bool

	CapturedAt      time.Time
	PeerDownStrikes int // hysteresis lives here, derived during refresh
}

// State holds the current view and the probes used to refresh it.
type State struct {
	cfg    *config.Config
	http   *probe.HTTPProbe
	pg     *probe.PostgresProbe
	valkey *probe.ValkeyProbe
	wg     *probe.WireGuardProbe
	icmp   *probe.ICMPProbe

	// atomic.Pointer makes reads lock-free and trivially correct;
	// writes still happen serially because only Refresh writes.
	current atomic.Pointer[Snapshot]
}

func New(cfg *config.Config, h *probe.HTTPProbe, pg *probe.PostgresProbe,
	v *probe.ValkeyProbe, wg *probe.WireGuardProbe, ic *probe.ICMPProbe) *State {
	s := &State{cfg: cfg, http: h, pg: pg, valkey: v, wg: wg, icmp: ic}
	empty := &Snapshot{}
	s.current.Store(empty)
	return s
}

// Refresh runs all probes concurrently and atomically updates current.
// Called once per reconciler tick; the goroutine fan-out is the same
// pattern from collectSnapshot.
func (s *State) Refresh(ctx context.Context) *Snapshot {
	var (
		wg   sync.WaitGroup
		snap = &Snapshot{CapturedAt: time.Now()}
	)

	// Fan out probes — same shape as before, but each probe call
	// is a method on a service in internal/probe rather than a
	// free function.
	wg.Add( /* count */ )
	go func() { defer wg.Done() /* snap.Vrrp = ... */ }()
	// ... etc

	wg.Wait()

	// Carry forward hysteresis state from the previous snapshot.
	prev := s.current.Load()
	if !snap.PeerWgReachable && !snap.PeerPhysReachable {
		snap.PeerDownStrikes = prev.PeerDownStrikes + 1
	}

	s.current.Store(snap)
	return snap
}

// Current returns the most recently refreshed snapshot. Lock-free read.
func (s *State) Current() *Snapshot {
	return s.current.Load()
}
