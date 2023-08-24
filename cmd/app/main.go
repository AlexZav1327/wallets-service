package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexZav1327/service/internal/httpserver"
	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/service"
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

	webServer := httpserver.NewServer("", 8080, service.NewWallet(pg, logger), logger)

	err = webServer.Run(ctx)
	if err != nil {
		logger.Panicf("webServer.Run: %s", err)
	}
}
