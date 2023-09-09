package xrserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AlexZav1327/service/models"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type Server struct {
	host    string
	port    int
	Server  *http.Server
	service RateService
	log     *logrus.Entry
}

type RateService interface {
	GetCurrentRate(to, from string) (models.ExchangeRate, error)
}

func New(host string, port int, service RateService, log *logrus.Logger) *Server {
	server := Server{
		host:    host,
		port:    port,
		log:     log.WithField("module", "xr_http"),
		service: service,
	}

	h := NewHandler(service, log)
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/xr", h.get)
	})

	server.Server = &http.Server{
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

		err := s.Server.Shutdown(shutdownCtx)
		if err != nil {
			s.log.Warningf("Server.Shutdown: %s", err)
		}
	}()

	err := s.Server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("Server.ListenAndServe: %w", err)
	}

	return nil
}
