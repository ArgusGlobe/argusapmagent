package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/argus/ecs-fargate-agent/internal/collector"
	"github.com/argus/ecs-fargate-agent/internal/config"
	"github.com/argus/ecs-fargate-agent/internal/grpcclient"
	"github.com/argus/ecs-fargate-agent/internal/health"
	"github.com/argus/ecs-fargate-agent/internal/identity"
	"github.com/argus/ecs-fargate-agent/internal/metadata"
	"github.com/argus/ecs-fargate-agent/internal/telemetry"
)

func main() {
	cfg := config.Load()

	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := health.Check(cfg.HealthURL()); err != nil {
			os.Exit(1)
		}
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var metaClient metadata.Client = metadata.NewClient(cfg.MetadataURI, http.DefaultClient, logger)
	task, err := metaClient.Task(ctx)
	if err != nil {
		logger.Warn("ecs metadata unavailable; switching to local mock mode", "error", err)
		metaClient = metadata.NewMockClient(logger)
		task, err = metaClient.Task(ctx)
		if err != nil {
			logger.Error("mock metadata unavailable", "error", err)
			os.Exit(1)
		}
	}

	agentID := identity.AgentID(task, time.Now().UTC())
	batcher := telemetry.NewBatcher(cfg, agentID, task)
	coll := collector.New(metaClient, task, cfg, logger)
	shipper := grpcclient.New(cfg, logger)

	healthServer := health.NewServer(cfg.HealthAddress(), logger)
	go func() {
		if err := healthServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("health server failed", "error", err)
		}
	}()

	if err := run(ctx, cfg, coll, batcher, shipper, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("agent stopped with error", "error", err)
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = healthServer.Stop(shutdownCtx)
	logger.Info("probe ecs fargate agent stopped")
}

func run(
	ctx context.Context,
	cfg config.Config,
	coll *collector.Collector,
	batcher *telemetry.Batcher,
	shipper *grpcclient.Client,
	logger *slog.Logger,
) error {
	logger.Info("probe ecs fargate agent started",
		"environment", cfg.Environment,
		"service", cfg.ServiceName,
		"interval", cfg.CollectionInterval,
	)

	ticker := time.NewTicker(cfg.CollectionInterval)
	defer ticker.Stop()

	send := func() {
		snapshot, err := coll.Collect(ctx)
		if err != nil {
			logger.Warn("collection failed", "error", err)
			return
		}
		batch := batcher.FromSnapshot(snapshot)
		if err := shipper.Send(ctx, batch); err != nil {
			logger.Warn("telemetry send failed", "error", err)
			return
		}
		logger.Debug("telemetry batch sent", "samples", len(batch.GetSamples()), "logs", len(batch.GetLogs()))
	}

	send()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			send()
		}
	}
}
