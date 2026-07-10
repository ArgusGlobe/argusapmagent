package telemetry

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/argus/ecs-fargate-agent/internal/config"
	pb "github.com/argus/ecs-fargate-agent/internal/hermespb"
	"github.com/argus/ecs-fargate-agent/internal/metadata"
)

type Batcher struct {
	cfg     config.Config
	agentID string
	task    metadata.Task
}

func NewBatcher(cfg config.Config, agentID string, task metadata.Task) *Batcher {
	return &Batcher{cfg: cfg, agentID: agentID, task: task}
}

func (b *Batcher) FromSnapshot(s Snapshot) *pb.Batch {
	out := &pb.Batch{
		AgentId:    b.agentID,
		Cluster:    clusterName(b.task.Cluster),
		Service:    serviceName(b.task, b.cfg.ServiceName),
		TaskArn:    b.task.TaskARN,
		LaunchType: launchType(b.task.LaunchType),
		SentAt:     timestamppb.New(time.Now().UTC()),
	}
	for _, sample := range s.Samples {
		out.Samples = append(out.Samples, &pb.ContainerSample{
			ContainerName: sample.ContainerName,
			CpuPercent:    sample.CPUPercent,
			MemUsageBytes: sample.MemUsageBytes,
			MemLimitBytes: sample.MemLimitBytes,
			MemPercent:    sample.MemPercent,
			NetRxBytes:    sample.NetRxBytes,
			NetTxBytes:    sample.NetTxBytes,
			BlkReadBytes:  sample.BlkReadBytes,
			BlkWriteBytes: sample.BlkWriteBytes,
			RestartCount:  int32(sample.RestartCount),
			Timestamp:     timestamppb.New(sample.Timestamp),
		})
	}
	for _, line := range s.Logs {
		out.Logs = append(out.Logs, &pb.LogLine{
			ContainerName: line.ContainerName,
			Stream:        line.Stream,
			Message:       line.Message,
			Timestamp:     timestamppb.New(line.Timestamp),
		})
	}
	return out
}

func launchType(raw string) pb.LaunchType {
	switch strings.ToUpper(raw) {
	case "FARGATE":
		return pb.LaunchType_LAUNCH_TYPE_FARGATE
	case "EC2":
		return pb.LaunchType_LAUNCH_TYPE_EC2
	default:
		return pb.LaunchType_LAUNCH_TYPE_UNSPECIFIED
	}
}

func clusterName(cluster string) string {
	if i := strings.LastIndex(cluster, "/"); i >= 0 && i+1 < len(cluster) {
		return cluster[i+1:]
	}
	return cluster
}

func serviceName(task metadata.Task, fallback string) string {
	if fallback = strings.TrimSpace(fallback); fallback != "" && fallback != "unknown-service" {
		return fallback
	}
	if task.Family != "" {
		return task.Family
	}
	return fallback
}
