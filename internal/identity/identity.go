package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/argus/ecs-fargate-agent/internal/metadata"
)

func AgentID(task metadata.Task, fallback time.Time) string {
	parts := []string{
		task.TaskARN,
		task.Cluster,
		serviceName(task),
	}
	if len(task.Containers) > 0 {
		parts = append(parts, task.Containers[0].DockerID)
	}
	if task.StartedAt != nil {
		parts = append(parts, task.StartedAt.UTC().Format(time.RFC3339Nano))
	} else if !fallback.IsZero() {
		parts = append(parts, fallback.UTC().Format(time.RFC3339Nano))
	}
	raw := strings.Join(nonEmpty(parts), "|")
	if raw == "" {
		return uuid.NewString()
	}
	sum := sha256.Sum256([]byte(raw))
	return "ecs-" + hex.EncodeToString(sum[:16])
}

func serviceName(task metadata.Task) string {
	arn := task.TaskARN
	if i := strings.Index(arn, "/"); i >= 0 {
		segments := strings.Split(arn[i+1:], "/")
		if len(segments) >= 2 {
			return segments[len(segments)-2]
		}
	}
	return task.Family
}

func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}
