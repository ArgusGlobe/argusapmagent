package metadata

import (
	"context"
	"log/slog"
	"time"
)

type MockClient struct {
	started time.Time
	logger  *slog.Logger
}

func NewMockClient(logger *slog.Logger) *MockClient {
	return &MockClient{started: time.Now().UTC(), logger: logger}
}

func (m *MockClient) Task(_ context.Context) (Task, error) {
	return Task{
		Cluster:          "local-dev-cluster",
		TaskARN:          "arn:aws:ecs:local:000000000000:task/local-dev-cluster/mock-task",
		Family:           "probe-local",
		Revision:         "1",
		DesiredStatus:    "RUNNING",
		KnownStatus:      "RUNNING",
		LaunchType:       "FARGATE",
		AvailabilityZone: "local-a",
		Limits:           Limits{CPU: 0.5, Memory: 1024},
		StartedAt:        &m.started,
		Containers: []Container{
			{
				DockerID:      "mock-app-container",
				Name:          "application",
				Image:         "probe/mock-app:latest",
				DesiredStatus: "RUNNING",
				KnownStatus:   "RUNNING",
				Limits:        Limits{CPU: 0.25, Memory: 512},
				StartedAt:     &m.started,
			},
			{
				DockerID:      "mock-probe-sidecar",
				Name:          "probe-sidecar",
				Image:         "probe/ecs-fargate-agent:local",
				DesiredStatus: "RUNNING",
				KnownStatus:   "RUNNING",
				Limits:        Limits{CPU: 0.25, Memory: 512},
				StartedAt:     &m.started,
			},
		},
	}, nil
}

func (m *MockClient) TaskStats(_ context.Context) (TaskStats, error) {
	now := time.Now().UTC()
	return TaskStats{
		"mock-app-container": {
			ID:        "mock-app-container",
			Name:      "application",
			Read:      now,
			PreRead:   now.Add(-30 * time.Second),
			Timestamp: now,
			CPUStats: CPUStats{
				CPUUsage:       CPUUsage{TotalUsage: 140_000_000},
				SystemCPUUsage: 1_000_000_000,
				OnlineCPUs:     2,
			},
			PreCPU: CPUStats{
				CPUUsage:       CPUUsage{TotalUsage: 100_000_000},
				SystemCPUUsage: 900_000_000,
				OnlineCPUs:     2,
			},
			Memory: MemoryStats{Usage: 180 * 1024 * 1024, Limit: 512 * 1024 * 1024},
			Networks: Networks{
				"eth0": {RxBytes: 10_240, TxBytes: 20_480, RxPackets: 120, TxPackets: 130},
			},
			BlkIO: BlkIOStats{IoServiceBytesRecursive: []BlkIOEntry{
				{Op: "Read", Value: 2048},
				{Op: "Write", Value: 4096},
			}},
		},
	}, nil
}
