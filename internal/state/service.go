package state

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/EHLO1/keel/internal/adapter/postgres"
)

// Check for pg volume, if it doesn't exist, ignore, if it does, look for standby.signal

type Service struct {
	current atomic.Pointer[Snapshot]
	pg      *postgres.Client
}

func NewService(pg *postgres.Client) (*Service, error) {
	s := &Service{pg: pg}
	empty := &Snapshot{}
	s.current.Store(empty)
	return s, nil
}

func (s *Service) Start(ctx context.Context) error {
	return nil
}

// Refresh uses all clients concurrently and atomically updates current.
// Called once per reconciler tick; the goroutine fan-out is the same
// pattern from collectSnapshot.
func (s *Service) Refresh(ctx context.Context) *Snapshot {
	var (
		wg   sync.WaitGroup
		snap = &Snapshot{CapturedAt: time.Now()}
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
func (s *Service) Current() *Snapshot {
	return s.current.Load()
}
