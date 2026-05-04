// Package probe contains the reachability probes Keel uses to build
// each tick's snapshot of reality. Each probe type lives in its own
// file: http.go (this file) for HTTP health checks, icmp.go for
// data-plane liveness over wg0, and so on.
package probe

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/EHLO1/keel/backend/internal/config"
)

// HTTPProbe performs HTTP GET reachability checks against a configurable
// set of endpoints (the load balancer health URL, the peer's queue-health
// endpoint). It owns a single *http.Client so connection pooling works as
// intended across consecutive probes.
//
// Construct one with NewHTTP, share it across the prober's lifetime, and
// call Check() per endpoint per tick.
type HTTPProbe struct {
	cfg    *config.Config
	client *http.Client
}

// NewHTTP constructs an HTTPProbe with sensible defaults for low-volume
// internal probing: short per-request timeout, small idle pool, no proxy.
//
// The client uses cfg.ProbeTimeout for both connect and total-request
// budget. Per-call deadlines via context.Context override this when
// stricter timing is needed.
func NewHTTP(cfg *config.Config) *HTTPProbe {
	transport := &http.Transport{
		// Small pool — we hit two endpoints, both internal.
		MaxIdleConns:        4,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     30 * time.Second,
		// Force-close the connection if the response headers don't
		// arrive within the probe budget. Prevents slow-loris-style
		// stalls from blocking the reconciliation tick.
		ResponseHeaderTimeout: cfg.ProbeTimeout,
		// Disable HTTP/2 explicitly — its multiplexing buys us
		// nothing for two endpoints, and HTTP/1.1's per-connection
		// state is easier to reason about during failures.
		ForceAttemptHTTP2: false,
	}
	return &HTTPProbe{
		cfg: cfg,
		client: &http.Client{
			Timeout:   cfg.ProbeTimeout,
			Transport: transport,
		},
	}
}

// Result is what each probe call returns. Latency is meaningful even
// on failure (it tells you how long we waited before giving up); status
// is zero on transport errors.
type Result struct {
	URL     string
	OK      bool
	Status  int
	Latency time.Duration
	Err     error
}

// Check performs one GET against url, returning OK only on a 2xx response.
// Honors ctx cancellation and the configured probe timeout, whichever is
// stricter.
func (p *HTTPProbe) Check(ctx context.Context, url string) Result {
	start := time.Now()
	r := Result{URL: url}

	cctx, cancel := context.WithTimeout(ctx, p.cfg.ProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		r.Err = fmt.Errorf("build request: %w", err)
		r.Latency = time.Since(start)
		return r
	}
	req.Header.Set("User-Agent", "keel/1.0")

	resp, err := p.client.Do(req)
	r.Latency = time.Since(start)
	if err != nil {
		r.Err = err
		return r
	}
	defer resp.Body.Close()

	r.Status = resp.StatusCode
	r.OK = resp.StatusCode >= 200 && resp.StatusCode < 300
	if !r.OK {
		r.Err = fmt.Errorf("non-2xx status: %d", resp.StatusCode)
	}
	return r
}

// Close releases connections held by the underlying transport. Call
// during graceful shutdown. Safe to call multiple times.
func (p *HTTPProbe) Close() {
	if t, ok := p.client.Transport.(*http.Transport); ok {
		t.CloseIdleConnections()
	}
}
