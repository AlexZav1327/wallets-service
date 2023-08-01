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
	service service.AccessData
}

func getCurrentTime() string {
	now := time.Now()

	return now.Format("02-01-2006 15:04:05")
}

func NewServer(host string, port int, service service.AccessData) *Server {
	server := Server{
		host:    host,
		port:    port,
		service: service,
	}

	r := chi.NewRouter()

	r.Get("/now", func(w http.ResponseWriter, r *http.Request) {
		service.SaveAccessData(r.RemoteAddr, getCurrentTime())

		if _, err := w.Write([]byte(service.ShowCurrentAccessData(r.RemoteAddr, getCurrentTime()))); err != nil {
			log.WithFields(log.Fields{
				"package":  "server",
				"function": "NewServer",
				"method":   "w.Write",
				"error":    err,
			}).Warning("Write current access data error")
		}
	})

	r.Get("/prev", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(service.ShowPreviousAccessData())); err != nil {
			log.WithFields(log.Fields{
				"package":  "server",
				"function": "NewServer",
				"method":   "w.Write",
				"error":    err,
			}).Warning("Write previous access data error")
		}
	})

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

		if err := s.server.Shutdown(shutdownCtx); err != nil {
			log.WithFields(log.Fields{
				"package":  "server",
				"function": "Run",
				"error":    err,
			}).Warning("Closing server")
		}
	}()

	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("Server starting error: %w", err)
	}

	return nil
}
