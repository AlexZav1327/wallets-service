package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/server"
	"github.com/AlexZav1327/service/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	defer cancel()

	dsn := os.Getenv("DSN")

	pg, err := postgres.ConnectDB(ctx, dsn)
	if err != nil {
		log.Panicf("postgres.ConnectDB(ctx, dsn): %s", err)
	}

	err = pg.Migrate(migrate.Up)
	if err != nil {
		log.Panicf("pg.Migrate(migrate.Up): %s", err)
	}

	webServer := server.NewServer("", 8083, service.NewAccessData(pg))

	err = webServer.Run(ctx)
	if err != nil {
		log.Panicf("webServer.Run(ctx): %s", err)
	}
}
