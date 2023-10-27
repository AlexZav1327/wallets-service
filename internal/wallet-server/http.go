package walletserver

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Server struct {
	host    string
	port    int
	Server  *http.Server
	service WalletService
	log     *logrus.Entry
	handler *Handler
}

func New(host string, port int, service WalletService, log *logrus.Logger, privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
) *Server {
	h := NewHandler(service, log, privateKey, publicKey)

	server := Server{
		host:    host,
		port:    port,
		service: service,
		log:     log.WithField("module", "http"),
		handler: h,
	}

	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	r.Group(func(r chi.Router) {
		r.Use(h.metric)
		r.Use(h.jwtAuth)
		r.Route("/api/v1", func(r chi.Router) {
			r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log, NoColor: true}))
			r.Post("/wallet/create", h.create)
			r.Get("/wallet/{id}", h.get)
			r.Get("/wallets", h.getList)
			r.Get("/wallet/history", h.getHistory)
			r.Patch("/wallet/{id}", h.update)
			r.Delete("/wallet/{id}", h.delete)
			r.Put("/wallet/{id}/deposit", h.deposit)
			r.Put("/wallet/{id}/withdraw", h.withdraw)
			r.Put("/wallet/{idSrc}/transfer/{idDst}", h.transfer)
		})
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

	defer s.log.Info("Server is stopped")

	go func() {
		<-ctx.Done()

		err := s.Server.Shutdown(shutdownCtx)
		if err != nil {
			s.log.Warningf("Server.Shutdown: %s", err)
		}
	}()

	s.log.Infof("Server is running at port %d...", s.port)

	err := s.Server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("Server.ListenAndServe: %w", err)
	}

	return nil
}

func (s *Server) GenerateToken(uuid, email string) (string, error) {
	return s.handler.generateToken(uuid, email)
}
