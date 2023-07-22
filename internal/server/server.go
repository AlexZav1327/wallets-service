package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AlexZav1327/service/internal/service"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	host    string
	port    int
	server  *http.Server
	service service.VisitInfo
}

func NewServer(host string, port int, service service.VisitInfo) *Server {
	server := Server{
		host:    host,
		port:    port,
		service: service,
	}

	r := chi.NewRouter()
	r.Get("/", service.ShowVisitInfo)
	r.Get("/prev", service.ShowPrevVisitInfo)

	server.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", host, port),
		Handler:           r,
		ReadHeaderTimeout: 30 * time.Second,
	}

	return &server
}

func (s *Server) Run(ctx context.Context) {
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

	defer cancel()

	go func() {
		<-ctx.Done()

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			log.WithFields(log.Fields{
				"package":  "server",
				"function": "Run",
				"error":    err,
			}).Warn("Closing server")
		}
	}()

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.WithFields(log.Fields{
			"package":  "server",
			"function": "Run",
			"error":    err,
		}).Panic("Server error")
	}
}
