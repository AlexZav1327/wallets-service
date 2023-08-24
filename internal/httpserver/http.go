package httpserver

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
	server  *http.Server
	service WalletService
	log     *logrus.Entry
}

type WalletService interface {
	Create(ctx context.Context, wallet models.WalletData) ([]models.WalletData, error)
	GetList(ctx context.Context) ([]models.WalletData, error)
	Get(ctx context.Context, id string) ([]models.WalletData, error)
	Update(ctx context.Context, id string, wallet models.WalletData) ([]models.WalletData, error)
	Delete(ctx context.Context, id string) error
}

func NewServer(host string, port int, service WalletService, log *logrus.Logger) *Server {
	server := Server{
		host:    host,
		port:    port,
		log:     log.WithField("module", "http"),
		service: service,
	}

	h := NewHandler(service, log)
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/create", h.create)
		r.Get("/wallets/{id}", h.get)
		r.Get("/wallets", h.getList)
		r.Patch("/wallets/{id}", h.update)
		r.Delete("/wallets/{id}", h.delete)
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
			s.log.Warningf("server.Shutdown: %s", err)
		}
	}()

	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server.ListenAndServe: %w", err)
	}

	return nil
}
