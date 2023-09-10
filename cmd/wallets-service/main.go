package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexZav1327/service/internal/postgres"
	walletserver "github.com/AlexZav1327/service/internal/wallet-server"
	walletservice "github.com/AlexZav1327/service/internal/wallet-service"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	logger := logrus.StandardLogger()
	dsn := os.Getenv("DSN")

	pg, err := postgres.ConnectDB(ctx, logger, dsn)
	if err != nil {
		logger.Panicf("postgres.ConnectDB: %s", err)
	}

	err = pg.Migrate(migrate.Up)
	if err != nil {
		logger.Panicf("Migrate: %s", err)
	}

	walletService := walletservice.New(pg, logger)
	server := walletserver.New("", 8080, walletService, logger)

	err = server.Run(ctx)
	if err != nil {
		logger.Panicf("server.Run: %s", err)
	}
}
