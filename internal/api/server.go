package api

import (
	"context"
	"net/http"
	"time"

	"github.com/EHLO1/keel/internal/state"
)

type Server struct {
	httpServer   *http.Server
	stateService *state.Service
	APIPort      string
}

func NewServer(port string, stateService *state.Service) *Server {
	s := &Server{
		stateService: stateService,
		APIPort:      port,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", s.handleGetState)

	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return s
}

func (s *Server) Start(ctx context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	}
}
