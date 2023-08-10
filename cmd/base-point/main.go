package main

import (
	"context"
	"os/signal"
	"sync"
	"syscall"

	"github.com/AlexZav1327/service/internal/postgres"
	"github.com/AlexZav1327/service/internal/server"
	"github.com/AlexZav1327/service/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
	log "github.com/sirupsen/logrus"
)

func main() {
	webServer := server.NewServer("", 8080, service.AccessData{})
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)

	dsn := "user=user password=1234 host=localhost port=5432 dbname=web-service sslmode=disable"

	go func() {
		defer wg.Done()

		err := webServer.Run(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"package":  "main",
				"function": "Run",
				"error":    err,
			}).Panic("failed to run server")
		}
	}()

	pg, err := postgres.ConnectDB(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to get postgres: %s", err)
	}

	if err = pg.Migrate(migrate.Up); err != nil {
		log.Fatalf("failed to migrate: %s", err)
	}

	wg.Wait()
}
