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
	"github.com/argus/ecs-fargate-agent/internal/otlp"
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
	var localOTLP *otlp.Receiver
	if cfg.OTLPEnabled {
		localOTLP = otlp.NewReceiver(cfg.OTLPGRPCAddr, telemetryServiceName(task, cfg.ServiceName), logger)
		if err := localOTLP.Start(ctx); err != nil {
			logger.Warn("probe local otlp receiver unavailable", "addr", cfg.OTLPGRPCAddr, "error", err)
		}
	}

	healthServer := health.NewServer(cfg.HealthAddress(), logger)
	go func() {
		if err := healthServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("health server failed", "error", err)
		}
	}()

	if err := run(ctx, cfg, coll, batcher, shipper, localOTLP, logger); err != nil && !errors.Is(err, context.Canceled) {
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
	localOTLP *otlp.Receiver,
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
		if localOTLP != nil {
			appMetrics, network, logs, rollups := localOTLP.Drain()
			snapshot.AppMetrics = append(snapshot.AppMetrics, appMetrics...)
			snapshot.Network = append(snapshot.Network, network...)
			snapshot.Logs = append(snapshot.Logs, logs...)
			snapshot.LogRollups = append(snapshot.LogRollups, rollups...)
		}
		batch := batcher.FromSnapshot(snapshot)
		if err := shipper.Send(ctx, batch); err != nil {
			logger.Warn("telemetry send failed", "error", err)
			return
		}
		logger.Debug("telemetry batch sent",
			"samples", len(batch.GetSamples()),
			"logs", len(batch.GetLogs()),
			"app_metrics", len(batch.GetAppMetrics()),
			"network", len(batch.GetNetwork()),
			"log_rollups", len(batch.GetLogRollups()),
		)
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

func telemetryServiceName(task metadata.Task, fallback string) string {
	if fallback != "" && fallback != "unknown-service" {
		return fallback
	}
	if task.Family != "" {
		return task.Family
	}
	return fallback
}
