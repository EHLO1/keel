package api

import (
	"context"
	"net/http"

	"github.com/EHLO1/keel/internal/app"
)

type Server struct {
	httpServer *http.Server
	state      *state.Service
}

func NewServer(cfg app.Config, stateService *state.Service) *Server {
	s := &Server{
		state: stateService,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", s.handleGetState)

	s.httpServer = &http.Server{
		Addr:    ":" + cfg.APIPort,
		Handler: mux,
	}

	return s
}

func (s *Server) Start(ctx context.Context) error {
	return nil
}
