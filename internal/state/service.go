package state

import (
	"context"
	"log/slog"
	"os"
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
	PG      *postgres.Client
	VK      *valkey.Client
	WRG     *wireguard.Client
	HC      *httpc.Client
	DR      *docker.Client
	IC      *icmp.Client
	NW      network.Client
	Sys     systemd.Client
	MM      *filesystem.MaintenanceFlag
	SS      *filesystem.StandbySignal
	VR      *filesystem.VRRPRole
	SF      *filesystem.StateFile
	SvcList []string
	Log     *slog.Logger
}

type Service struct {
	current atomic.Pointer[Snapshot]

	// Clients
	pg      *postgres.Client
	vk      *valkey.Client
	wrg     *wireguard.Client
	hc      *httpc.Client
	dr      *docker.Client
	ic      *icmp.Client
	nw      network.Client
	sys     systemd.Client
	mm      *filesystem.MaintenanceFlag
	ss      *filesystem.StandbySignal
	vr      *filesystem.VRRPRole
	sf      *filesystem.StateFile
	svcList []string
	log     *slog.Logger
}

func NewService(deps Dependencies) (*Service, error) {
	s := &Service{
		pg:      deps.PG,
		vk:      deps.VK,
		wrg:     deps.WRG,
		hc:      deps.HC,
		dr:      deps.DR,
		ic:      deps.IC,
		nw:      deps.NW,
		sys:     deps.Sys,
		mm:      deps.MM,
		ss:      deps.SS,
		vr:      deps.VR,
		sf:      deps.SF,
		svcList: deps.SvcList,
		log:     deps.Log,
	}

	// Initialize atomic pointer with an empty snapshot
	s.current.Store(&Snapshot{})
	return s, nil
}

func (s *Service) Capture(parentCtx context.Context, log *slog.Logger) *Snapshot {
	ctx, cancel := context.WithTimeout(parentCtx, 500*time.Millisecond)
	defer cancel()

	wrgConfig := s.wrg.Observe()
	hostname, _ := os.Hostname()
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		snap = &Snapshot{
			CapturedAt:     time.Now(),
			Hostname:       hostname,
			WireguardState: wrgConfig,
		}
	)

	for i, peer := range wrgConfig.Peers {
		snap.PeerKeelInstances[i] = PeerKeelInstance{
			WireguardIP: peer.AllowedIPs[0],
			RealIP:      peer.Endpoint,
		}
	}

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
		ss := s.sys.Observe(ctx, s.svcList)
		assign(func() { snap.Systemd = ss })
	})

	for i, peerInst := range snap.PeerKeelInstances {

		wg.Go(func() {
			err := s.ic.Ping(ctx, 300*time.Millisecond, peerInst.WireguardIP)
			assign(func() {
				snap.PeerKeelInstances[i].PingableOverWireguard = (err == nil)
			})
		})

		wg.Go(func() {
			err := s.ic.Ping(ctx, 300*time.Millisecond, peerInst.RealIP)
			assign(func() {
				snap.PeerKeelInstances[i].PingableOverReal = (err == nil)
			})
		})

		wg.Go(func() {
			peerSnap, reachable := s.hc.ObservePeerKeelStateAPI(ctx, peerInst.WireguardIP)
			assign(func() {
				snap.PeerKeelInstances[i].APISnapshot = peerSnap
				snap.PeerKeelInstances[i].APISnapshotAgeSeconds = time.Since(peerSnap.CapturedAt).Seconds()
				snap.PeerKeelInstances[i].APIReachable = reachable
			})
		})
	}

	wg.Go(func() {
		p := s.pg.Observe(ctx)
		assign(func() { snap.Postgres = p })
	})

	wg.Go(func() {
		v := s.vk.Observe(ctx)
		assign(func() { snap.Valkey = v })
	})

	wg.Go(func() {
		o, err := s.nw.ObserveVIPOwnership()
		if err != nil {
			assign(func() { snap.OwnsVIP = false })
		}
		assign(func() { snap.OwnsVIP = o })
	})

	wg.Go(func() {
		r, err := s.vr.Observe()
		if err != nil {
			assign(func() { snap.VRRPRole = "UNKNOWN" })
		}
		assign(func() { snap.VRRPRole = r })
	})

	wg.Go(func() {
		stf, err := s.sf.Observe()
		if err != nil {
			assign(func() { snap.LocalState = LocalUnknown })
		}
		assign(func() {
			switch stf {
			case "PRIMARY":
				snap.LocalState = LocalPrimary
			case "STANDBY":
				snap.LocalState = LocalStandby
			default:
				snap.LocalState = LocalUnhealthy
			}
		})
	})

	wg.Go(func() {
		m, err := s.mm.Observe()
		if err != nil {
			assign(func() { snap.MaintenanceMode = m })
		}
		assign(func() { snap.MaintenanceMode = m })
	})

	wg.Go(func() {
		pgVolPath, err := s.dr.GetVolumeMountpoint(ctx)
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
