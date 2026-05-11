package state

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/EHLO1/keel/internal/types"
)

type StateService struct {
	current atomic.Pointer[types.Snapshot]
}

func NewService() (*StateService, error) {
	s := &StateService{}
	empty := &types.Snapshot{}
	s.current.Store(empty)
	return s, nil
}

func (s *StateService) Start(ctx context.Context) error {
	return nil
}

// Refresh uses all clients concurrently and atomically updates current.
// Called once per reconciler tick; the goroutine fan-out is the same
// pattern from collectSnapshot.
func (s *StateService) Refresh(ctx context.Context) *types.Snapshot {
	var (
		wg   sync.WaitGroup
		snap = &types.Snapshot{CapturedAt: time.Now()}
	)

	// Fan out probes — same shape as before, but each probe call
	// is a method on a StateService in internal/probe rather than a
	// free function.
	wg.Add(1)
	go func() { defer wg.Done() /* snap.Vrrp = ... */ }()
	// ... etc

	wg.Wait()

	// Carry forward hysteresis state from the previous snapshot.
	prev := s.current.Load()
	if !snap.PeerIsReachableWireGuard && !snap.PeerIsReachable {
		snap.PeerDownStrikes = prev.PeerDownStrikes + 1
	}

	s.current.Store(snap)
	return snap
}

// Current returns the most recently refreshed snapshot. Lock-free read.
func (s *StateService) Current() *types.Snapshot {
	return s.current.Load()
}
