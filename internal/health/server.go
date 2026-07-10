package health

import (
	"context"
	"log/slog"
	"net/http"
)

type Server struct {
	server *http.Server
	logger *slog.Logger
}

func NewServer(addr string, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return &Server{server: &http.Server{Addr: addr, Handler: mux}, logger: logger}
}

func (s *Server) Start() error {
	s.logger.Info("health endpoint listening", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
