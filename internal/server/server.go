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
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Panicf("time.LoadLocation(): %s", err)
	}

	now := time.Now().In(location)

	return now.Format("2006-01-02 15:04:05")
}

func NewServer(host string, port int, service *service.AccessData) *Server {
	server := Server{
		host:    host,
		port:    port,
		service: *service,
	}

	r := chi.NewRouter()

	r.Get("/now", func(w http.ResponseWriter, r *http.Request) {
		err := service.SaveAccessData(r.Context(), r.RemoteAddr, getCurrentTime())
		if err != nil {
			log.Panicf("service.SaveAccessData(): %s", err)
		}

		_, err = w.Write([]byte(service.ShowCurrentAccessData(r.RemoteAddr, getCurrentTime())))
		if err != nil {
			log.Panicf("w.Write([]byte(service.ShowCurrentAccessData())): %s", err)
		}
	})

	r.Get("/prev", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(service.ShowPreviousAccessData(r.Context())))
		if err != nil {
			log.Panicf("w.Write([]byte(service.ShowPreviousAccessData())): %s", err)
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

		err := s.server.Shutdown(shutdownCtx)
		if err != nil {
			log.Warningf("s.server.Shutdown(shutdownCtx): %s", err)
		}
	}()

	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("s.server.ListenAndServe(): %w", err)
	}

	return nil
}
