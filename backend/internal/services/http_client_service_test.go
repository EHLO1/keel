package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EHLO1/keel/backend/internal/config"
)

func newTestProbe() *HTTPClientService {
	return NewHTTPClientService(&config.Config{ProbeTimeout: 2 * time.Second})
}

func TestCheck_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := newTestProbe()
	defer p.Close()

	r := p.Check(context.Background(), srv.URL)
	if !r.OK {
		t.Fatalf("expected OK, got err=%v status=%d", r.Err, r.Status)
	}
	if r.Status != 200 {
		t.Errorf("expected 200, got %d", r.Status)
	}
	if r.Latency <= 0 {
		t.Errorf("expected positive latency, got %v", r.Latency)
	}
}

func TestCheck_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	p := newTestProbe()
	defer p.Close()

	r := p.Check(context.Background(), srv.URL)
	if r.OK {
		t.Errorf("expected !OK on 503")
	}
	if r.Status != 503 {
		t.Errorf("expected 503, got %d", r.Status)
	}
	if r.Err == nil {
		t.Errorf("expected error on non-2xx")
	}
}

func TestCheck_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Tighter timeout than the server's response delay.
	p := NewHTTPClientService(&config.Config{ProbeTimeout: 50 * time.Millisecond})
	defer p.Close()

	r := p.Check(context.Background(), srv.URL)
	if r.OK {
		t.Errorf("expected timeout, got OK")
	}
	if r.Err == nil {
		t.Errorf("expected timeout error")
	}
}

func TestCheck_BadURL(t *testing.T) {
	p := newTestProbe()
	defer p.Close()

	r := p.Check(context.Background(), "http://127.0.0.1:1/nope")
	if r.OK {
		t.Errorf("expected failure on unreachable port")
	}
	if r.Err == nil {
		t.Errorf("expected error")
	}
}

func TestCheck_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	p := newTestProbe()
	defer p.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	r := p.Check(ctx, srv.URL)
	if r.OK {
		t.Errorf("expected cancellation to fail the check")
	}
}

func TestCheck_SetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := newTestProbe()
	defer p.Close()

	p.Check(context.Background(), srv.URL)
	if gotUA != "keel/1.0" {
		t.Errorf("expected keel/1.0 user agent, got %q", gotUA)
	}
}
