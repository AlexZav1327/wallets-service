package main

import (
	"context"
	"crypto/rsa"
	_ "embed"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/rates"
	walletserver "github.com/AlexZav1327/service/internal/wallet-server"
	walletservice "github.com/AlexZav1327/service/internal/wallet-service"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
)

const (
	port = 8080
	host = ""
)

//go:embed private.pem
var embedSigningKey string

//go:embed public.pem
var embedVerificationKey string

func main() {
	var (
		pgDSN           = getEnv(os.Getenv("PG_DSN"), "postgres://user:secret@localhost:5432/postgres?sslmode=disable")
		signingKey      = getEnv(os.Getenv("PRIVATE_SIGNING_KEY"), embedSigningKey)
		verificationKey = getEnv(os.Getenv("PUBLIC_VERIFICATION_KEY"), embedVerificationKey)
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	logger := logrus.StandardLogger()

	pg, err := postgres.ConnectDB(ctx, logger, pgDSN)
	if err != nil {
		logger.Panicf("postgres.ConnectDB: %s", err)
	}

	err = pg.Migrate(migrate.Up)
	if err != nil {
		logger.Panicf("Migrate: %s", err)
	}

	exchangeRates := rates.New(logger)
	walletsService := walletservice.New(pg, exchangeRates, logger)
	server := walletserver.New(
		host,
		port,
		walletsService,
		logger,
		mustGetPrivateKey(signingKey),
		mustGetPublicKey(verificationKey),
	)

	err = walletsService.TrackerRun(ctx)
	if err != nil {
		logger.Panicf("walletsService.TrackerRun: %s", err)
	}

	err = server.Run(ctx)
	if err != nil {
		logger.Panicf("server.Run: %s", err)
	}
}

func getEnv(env, defaultValue string) string {
	value := os.Getenv(env)
	if value == "" {
		return defaultValue
	}

	return value
}

func mustGetPrivateKey(key string) *rsa.PrivateKey {
	if len(key) == 0 {
		logrus.Panic("File public.pem is missing or invalid")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(key))
	if err != nil {
		logrus.Panicf("jwt.ParseRSAPrivateKeyFromPEM: %s", err)
	}

	return privateKey
}

func mustGetPublicKey(key string) *rsa.PublicKey {
	if len(key) == 0 {
		logrus.Panic("File public.pem is missing or invalid")
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(key))
	if err != nil {
		logrus.Panicf("jwt.ParseRSAPublicKeyFromPEM: %s", err)
	}

	return publicKey
}
