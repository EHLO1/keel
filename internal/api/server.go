package api

import (
	"context"
	"net/http"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(port string) *Server {
	s := &Server{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", s.handleGetState)

	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return s
}

func (s *Server) Start(ctx context.Context) error {
	return nil
}
