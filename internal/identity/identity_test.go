package identity

import (
	"testing"
	"time"

	"github.com/argus/ecs-fargate-agent/internal/metadata"
)

func TestAgentIDStable(t *testing.T) {
	start := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	task := metadata.Task{
		TaskARN:   "arn:aws:ecs:ap-south-1:1:task/cluster/service/abc",
		Cluster:   "arn:aws:ecs:ap-south-1:1:cluster/cluster",
		Family:    "svc",
		StartedAt: &start,
		Containers: []metadata.Container{
			{DockerID: "container-1"},
		},
	}
	a := AgentID(task, time.Now())
	b := AgentID(task, time.Now().Add(time.Hour))
	if a != b {
		t.Fatalf("expected stable agent id, got %q and %q", a, b)
	}
}
