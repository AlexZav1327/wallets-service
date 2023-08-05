package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/AlexZav1327/service/internal/server"
	"github.com/AlexZav1327/service/internal/service"
	log "github.com/sirupsen/logrus"
)

func main() {
	server := server.NewServer("", 8080, service.AccessData{})
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	defer cancel()

	err := server.Run(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"package":  "main",
			"function": "Run",
			"error":    err,
		}).Panic()
	}
}
