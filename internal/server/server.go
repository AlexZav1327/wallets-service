package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AlexZav1327/service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type Server struct {
	host    string
	port    int
	server  *http.Server
	service service.AccessData
	log     *logrus.Entry
}

func NewServer(host string, port int, service *service.AccessData, log *logrus.Logger) *Server {
	server := Server{
		host:    host,
		port:    port,
		log:     log.WithField("module", "server"),
		service: *service,
	}

	h := NewHandler(service, log)
	r := chi.NewRouter()

	r.Get("/now", h.current)
	r.Get("/prev", h.previous)

	server.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", host, port),
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
	}

	return &server
}

func (s *Server) Run(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	go func() {
		<-ctx.Done()

		err := s.server.Shutdown(shutdownCtx)
		if err != nil {
			s.log.Warningf("server.Shutdown(shutdownCtx): %s", err)
		}
	}()

	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server.ListenAndServe(): %w", err)
	}

	return nil
}
