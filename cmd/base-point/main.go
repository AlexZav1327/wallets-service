package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/AlexZav1327/service/internal/server"
	"github.com/AlexZav1327/service/internal/service"
)

func main() {
	server := server.NewServer("localhost", 8088, service.VisitInfo{})
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	defer cancel()

	server.Run(ctx)
	<-ctx.Done()
}
