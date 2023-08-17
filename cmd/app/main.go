package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/server"
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
		log.Panicf("postgres.ConnectDB(ctx, dsn): %s", err)
	}

	err = pg.Migrate(migrate.Up)
	if err != nil {
		log.Panicf("Migrate(migrate.Up): %s", err)
	}

	webServer := server.NewServer("", 8080, service.NewAccessData(pg, logger), logger)

	err = webServer.Run(ctx)
	if err != nil {
		log.Panicf("webServer.Run(ctx): %s", err)
	}
}
