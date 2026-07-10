package metadata

import (
	"context"
	"time"
)

type Client interface {
	Task(ctx context.Context) (Task, error)
	TaskStats(ctx context.Context) (TaskStats, error)
}

type Task struct {
	Cluster          string      `json:"Cluster"`
	TaskARN          string      `json:"TaskARN"`
	Family           string      `json:"Family"`
	Revision         string      `json:"Revision"`
	DesiredStatus    string      `json:"DesiredStatus"`
	KnownStatus      string      `json:"KnownStatus"`
	LaunchType       string      `json:"LaunchType"`
	AvailabilityZone string      `json:"AvailabilityZone"`
	Limits           Limits      `json:"Limits"`
	Containers       []Container `json:"Containers"`
	PullStartedAt    *time.Time  `json:"PullStartedAt"`
	PullStoppedAt    *time.Time  `json:"PullStoppedAt"`
	CreatedAt        *time.Time  `json:"CreatedAt"`
	StartedAt        *time.Time  `json:"StartedAt"`
	StoppedAt        *time.Time  `json:"StoppedAt"`
}

type Limits struct {
	CPU    float64 `json:"CPU"`
	Memory float64 `json:"Memory"`
}

type Container struct {
	DockerID      string     `json:"DockerId"`
	Name          string     `json:"Name"`
	DockerName    string     `json:"DockerName"`
	Image         string     `json:"Image"`
	ImageID       string     `json:"ImageID"`
	DesiredStatus string     `json:"DesiredStatus"`
	KnownStatus   string     `json:"KnownStatus"`
	Limits        Limits     `json:"Limits"`
	CreatedAt     *time.Time `json:"CreatedAt"`
	StartedAt     *time.Time `json:"StartedAt"`
	FinishedAt    *time.Time `json:"FinishedAt"`
}

type TaskStats map[string]ContainerStats

type ContainerStats struct {
	Read      time.Time   `json:"read"`
	PreRead   time.Time   `json:"preread"`
	CPUStats  CPUStats    `json:"cpu_stats"`
	PreCPU    CPUStats    `json:"precpu_stats"`
	Memory    MemoryStats `json:"memory_stats"`
	Networks  Networks    `json:"networks"`
	BlkIO     BlkIOStats  `json:"blkio_stats"`
	Name      string      `json:"name"`
	ID        string      `json:"id"`
	Timestamp time.Time   `json:"-"`
}

type CPUStats struct {
	CPUUsage       CPUUsage      `json:"cpu_usage"`
	SystemCPUUsage uint64        `json:"system_cpu_usage"`
	OnlineCPUs     uint32        `json:"online_cpus"`
	Throttling     ThrottleStats `json:"throttling_data"`
}

type CPUUsage struct {
	TotalUsage        uint64   `json:"total_usage"`
	PercpuUsage       []uint64 `json:"percpu_usage"`
	UsageInKernelmode uint64   `json:"usage_in_kernelmode"`
	UsageInUsermode   uint64   `json:"usage_in_usermode"`
}

type ThrottleStats struct {
	Periods          uint64 `json:"periods"`
	ThrottledPeriods uint64 `json:"throttled_periods"`
	ThrottledTime    uint64 `json:"throttled_time"`
}

type MemoryStats struct {
	Usage    uint64            `json:"usage"`
	MaxUsage uint64            `json:"max_usage"`
	Limit    uint64            `json:"limit"`
	Stats    map[string]uint64 `json:"stats"`
}

type Networks map[string]NetworkStats

type NetworkStats struct {
	RxBytes   uint64 `json:"rx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxBytes   uint64 `json:"tx_bytes"`
	TxPackets uint64 `json:"tx_packets"`
}

type BlkIOStats struct {
	IoServiceBytesRecursive []BlkIOEntry `json:"io_service_bytes_recursive"`
}

type BlkIOEntry struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
	Op    string `json:"op"`
	Value uint64 `json:"value"`
}
