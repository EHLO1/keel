package state

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/EHLO1/keel/internal/adapter/docker"
	"github.com/EHLO1/keel/internal/adapter/filesystem"
	"github.com/EHLO1/keel/internal/adapter/httpc"
	"github.com/EHLO1/keel/internal/adapter/icmp"
	"github.com/EHLO1/keel/internal/adapter/network"
	"github.com/EHLO1/keel/internal/adapter/postgres"
	"github.com/EHLO1/keel/internal/adapter/systemd"
	"github.com/EHLO1/keel/internal/adapter/valkey"
	"github.com/EHLO1/keel/internal/adapter/wireguard"
)

type Dependencies struct {
	PG        *postgres.Client
	VK        *valkey.Client
	WireGuard *wireguard.Client
	HTTPC     *httpc.Client
	Docker    *docker.Client
	ICMP      *icmp.Client
	Network   network.Client
	Sys       systemd.Client
	MM        *filesystem.MaintenanceFlag
	SS        *filesystem.StandbySignal
	VR        *filesystem.VRRPRole
	Log       *slog.Logger
}

type Service struct {
	current atomic.Pointer[Snapshot]

	// Clients
	pg        *postgres.Client
	vk        *valkey.Client
	wireguard *wireguard.Client
	httpC     *httpc.Client
	docker    *docker.Client
	icmp      *icmp.Client
	network   network.Client
	sys       systemd.Client
	mm        *filesystem.MaintenanceFlag
	ss        *filesystem.StandbySignal
	vr        *filesystem.VRRPRole
	log       *slog.Logger
}

func NewService(deps Dependencies) (*Service, error) {
	s := &Service{
		pg:        deps.PG,
		vk:        deps.VK,
		wireguard: deps.WireGuard,
		httpC:     deps.HTTPC,
		docker:    deps.Docker,
		icmp:      deps.ICMP,
		network:   deps.Network,
		sys:       deps.Sys,
		mm:        deps.MM,
		ss:        deps.SS,
		vr:        deps.VR,
		log:       deps.Log,
	}

	// Initialize atomic pointer with an empty snapshot
	s.current.Store(&Snapshot{})
	return s, nil
}

func (s *Service) Capture(parentCtx context.Context, log *slog.Logger) *Snapshot {
	ctx, cancel := context.WithTimeout(parentCtx, 500*time.Millisecond)
	defer cancel()

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		snap = &Snapshot{CapturedAt: time.Now()}
	)

	assign := func(update func()) {
		// Only write to the snapshot if the timeout hasn't hit.
		// If ctx.Err() != nil, the 500ms passed and Reconciler is already reading this struct.
		mu.Lock()
		defer mu.Unlock()
		if ctx.Err() == nil {
			update()
		}
	}

	wg.Go(func() {
		ss := s.sys.Observe(ctx, []string{"docker.service", "keepalived.service", "wireguard.service"})
		assign(func() { snap.Systemd = ss })
	})

	wg.Go(func() {
		p := s.pg.Observe(ctx)
		assign(func() { snap.Postgres = p })
	})

	wg.Go(func() {
		v := s.vk.Observe(ctx)
		assign(func() { snap.Valkey = v })
	})

	wg.Go(func() {
		o, err := s.network.ObserveVIPOwnership()
		if err != nil {
			assign(func() { snap.OwnsVIP = false })
		}
		assign(func() { snap.OwnsVIP = o })
	})

	wg.Go(func() {
		w := s.wireguard.Observe()
		assign(func() { snap.WireGuardHandshakeStatus = w })
	})

	wg.Go(func() {
		t := s.icmp.Observe(ctx, 300*time.Millisecond)
		assign(func() { snap.ICMPTargets = t })
	})

	wg.Go(func() {
		t := s.icmp.Observe(ctx, 300*time.Millisecond)
		assign(func() { snap.ICMPTargets = t })
	})

	wg.Go(func() {
		r, err := s.vr.Observe()
		if err != nil {
			assign(func() { snap.VRRPRole = "UNKNOWN" })
		}
		assign(func() { snap.VRRPRole = r })
	})

	wg.Go(func() {
		m, err := s.mm.Observe()
		if err != nil {
			assign(func() { snap.MaintenanceMode = m })
		}
		assign(func() { snap.MaintenanceMode = m })
	})

	wg.Go(func() {
		pgVolPath, err := s.docker.GetVolumeMountpoint(ctx)
		if err != nil {
			assign(func() { snap.PostgresInStandby = false })
		} else {
			stbySig, err := s.ss.Observe(pgVolPath)
			if err != nil {
				assign(func() { snap.PostgresInStandby = stbySig })
			}
			assign(func() { snap.PostgresInStandby = stbySig })
		}
	})

	// rest of clients

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-ctx.Done():
		// Timeout elapsed. Return what finished.
		s.log.Warn("state snapshot timed out, incomplete snapshot delivered", "duration", time.Since(snap.CapturedAt))
	}

	s.current.Store(snap)
	return snap
}

// Current returns the most recently refreshed snapshot. Lock-free read.
func (s *Service) Current() *Snapshot {
	return s.current.Load()
}
