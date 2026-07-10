package collector

import (
	"testing"

	"github.com/argus/ecs-fargate-agent/internal/metadata"
)

func TestCPUPercent(t *testing.T) {
	got := CPUPercent(metadata.ContainerStats{
		CPUStats: metadata.CPUStats{
			CPUUsage:       metadata.CPUUsage{TotalUsage: 140},
			SystemCPUUsage: 1000,
			OnlineCPUs:     2,
		},
		PreCPU: metadata.CPUStats{
			CPUUsage:       metadata.CPUUsage{TotalUsage: 100},
			SystemCPUUsage: 900,
			OnlineCPUs:     2,
		},
	})
	if got != 80 {
		t.Fatalf("expected 80, got %f", got)
	}
}
