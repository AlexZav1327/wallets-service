//nolint:wrapcheck
package main

import (
	"context"
	"crypto/rsa"
	_ "embed"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexZav1327/service/internal/messages"
	"github.com/AlexZav1327/service/internal/notifications"
	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/rates"
	walletserver "github.com/AlexZav1327/service/internal/wallet-server"
	walletservice "github.com/AlexZav1327/service/internal/wallet-service"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

//go:embed private.pem
var embedSigningKey string

//go:embed public.pem
var embedVerificationKey string

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath("./config")

	if err := viper.BindEnv("database.dsn", "PG_DSN"); err != nil {
		logrus.Warningf("viper.BindEnv: %s", err)
	}

	if err := viper.ReadInConfig(); err != nil {
		logrus.Panicf("viper.ReadInConfig: %s", err)
	}

	var (
		pgDSN           = viper.GetString("database.dsn")
		host            = viper.GetString("server.host")
		port            = viper.GetInt("server.port")
		signingKey      = getEnv("PRIVATE_SIGNING_KEY", embedSigningKey)
		verificationKey = getEnv("PUBLIC_VERIFICATION_KEY", embedVerificationKey)
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	logger := logrus.StandardLogger()

	pg, err := postgres.ConnectDB(ctx, logger, pgDSN)
	if err != nil {
		logger.Panicf("postgres.ConnectDB: %s", err)
	}

	if err = pg.Migrate(migrate.Up); err != nil {
		logger.Panicf("Migrate: %s", err)
	}

	exchangeRates := rates.New(logger)
	message := messages.New(logger)
	notification := notifications.New(logger)
	walletsService := walletservice.New(pg, exchangeRates, message, notification, logger)
	server := walletserver.New(
		host,
		port,
		walletsService,
		logger,
		mustGetPrivateKey(signingKey),
		mustGetPublicKey(verificationKey),
	)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return server.Run(ctx)
	})

	eg.Go(func() error {
		return walletsService.TrackerRun(ctx)
	})

	if err = eg.Wait(); err != nil {
		logrus.Panicf("eg.Wait: %s", err)
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
