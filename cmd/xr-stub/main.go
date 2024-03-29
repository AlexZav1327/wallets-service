package main

import (
	"context"
	"os/signal"
	"syscall"

	xrserver "github.com/AlexZav1327/service/internal/xr-server"
	xrservice "github.com/AlexZav1327/service/internal/xr-service"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	logger := logrus.StandardLogger()
	rateService := xrservice.New(logger)
	server := xrserver.New("", 8091, rateService, logger)

	if err := server.Run(ctx); err != nil {
		logger.Panicf("server.Run: %s", err)
	}
}
