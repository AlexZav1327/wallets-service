package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AlexZav1327/service/internal/postgres"

	"github.com/AlexZav1327/service/internal/service"
	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	host    string
	port    int
	server  *http.Server
	service service.AccessData
	pg      postgres.Postgres
}

func getCurrentTime() string {
	now := time.Now()

	return now.Format("02-01-2006 15:04:05")
}

func NewServer(host string, port int, service service.AccessData, pg *postgres.Postgres) *Server {
	server := Server{
		host:    host,
		port:    port,
		service: service,
		pg:      *pg,
	}

	r := chi.NewRouter()

	r.Get("/now", func(w http.ResponseWriter, r *http.Request) {
		if err := pg.StoreAccessData(r.RemoteAddr, getCurrentTime()); err != nil {
			log.WithFields(log.Fields{
				"package":  "server",
				"function": "NewServer",
				"error":    err,
			}).Error("Unable to store current access data in the database")
		}

		if _, err := w.Write([]byte(service.ShowCurrentAccessData(r.RemoteAddr, getCurrentTime()))); err != nil {
			log.WithFields(log.Fields{
				"package":  "server",
				"function": "NewServer",
				"error":    err,
			}).Warning("Unable to show current access data")
		}
	})

	r.Get("/prev", func(w http.ResponseWriter, r *http.Request) {
		data, err := pg.FetchAccessData()
		if err != nil {
			log.WithFields(log.Fields{
				"package":  "server",
				"function": "NewServer",
				"error":    err,
			}).Error("Unable to fetch current access data from the database")
		}

		if _, err := w.Write([]byte(service.ShowPreviousAccessData(data))); err != nil {
			log.WithFields(log.Fields{
				"package":  "server",
				"function": "NewServer",
				"error":    err,
			}).Warning("Unable to show previous access data")
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
		log.WithFields(log.Fields{
			"package":  "server",
			"function": "Run",
			"error":    err,
		}).Error("Unable to start server")
	}

	return nil
}
