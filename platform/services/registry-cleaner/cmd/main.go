package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kbntx-org/nexus/platform/services/registry-cleaner/internal/config"
	"github.com/kbntx-org/nexus/platform/services/registry-cleaner/internal/dockerhub"
	"github.com/kbntx-org/nexus/platform/services/registry-cleaner/internal/logger"
	"github.com/kbntx-org/nexus/platform/services/registry-cleaner/internal/retention"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	appConfig, err := config.Load()
	if err != nil {
		return err
	}
	slog.SetDefault(logger.New(appConfig.LogLevel))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	registry := dockerhub.NewClient(
		dockerhub.DefaultBaseURL,
		appConfig.DockerHub.Namespace,
		appConfig.DockerHub.Username,
		appConfig.DockerHub.Token,
	)

	service := retention.NewService(registry, appConfig.Policies, appConfig.DryRun, time.Now)
	return service.Run(ctx)
}
