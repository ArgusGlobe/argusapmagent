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
			Severity:      line.Severity,
			TraceId:       line.TraceID,
			SpanId:        line.SpanID,
			Labels:        labelsToProto(line.Labels),
		})
	}
	for _, metric := range s.AppMetrics {
		out.AppMetrics = append(out.AppMetrics, &pb.AppMetric{
			ContainerName: metric.ContainerName,
			Namespace:     metric.Namespace,
			Name:          metric.Name,
			Value:         metric.Value,
			Unit:          metric.Unit,
			Labels:        labelsToProto(metric.Labels),
			Timestamp:     timestamppb.New(metric.Timestamp),
		})
	}
	for _, call := range s.Network {
		out.Network = append(out.Network, &pb.NetworkCall{
			ContainerName: call.ContainerName,
			Direction:     call.Direction,
			Protocol:      call.Protocol,
			Method:        call.Method,
			Peer:          call.Peer,
			Host:          call.Host,
			Path:          call.Path,
			StatusCode:    int32(call.StatusCode),
			DurationMs:    call.DurationMS,
			BytesIn:       call.BytesIn,
			BytesOut:      call.BytesOut,
			Error:         call.Error,
			Timestamp:     timestamppb.New(call.Timestamp),
		})
	}
	for _, rollup := range s.LogRollups {
		out.LogRollups = append(out.LogRollups, &pb.LogRollup{
			ContainerName: rollup.ContainerName,
			Stream:        rollup.Stream,
			Level:         rollup.Level,
			Fingerprint:   rollup.Fingerprint,
			Count:         rollup.Count,
			SampleMessage: rollup.SampleMessage,
			FirstSeen:     timestamppb.New(rollup.FirstSeen),
			LastSeen:      timestamppb.New(rollup.LastSeen),
		})
	}
	return out
}

func labelsToProto(labels map[string]string) []*pb.Label {
	if len(labels) == 0 {
		return nil
	}
	out := make([]*pb.Label, 0, len(labels))
	for k, v := range labels {
		out = append(out, &pb.Label{Key: k, Value: v})
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
