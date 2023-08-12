package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	migrate "github.com/rubenv/sql-migrate"

	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/server"
	"github.com/AlexZav1327/service/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	defer cancel()

	dsn := os.Getenv("DSN")

	pg, err := postgres.ConnectDB(ctx, dsn)
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "main",
			"function": "ConnectDB",
			"error":    err,
		}).Error("Unable to get postgres")
	}

	if err = pg.Migrate(migrate.Up); err != nil {
		log.WithFields(log.Fields{
			"package":  "main",
			"function": "Migrate",
			"error":    err,
		}).Error("Unable to migrate")
	}

	webServer := server.NewServer("", 8080, service.AccessData{}, pg)

	err = webServer.Run(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "main",
			"function": "Run",
			"error":    err,
		}).Error("Unable to run server")
	}
}
