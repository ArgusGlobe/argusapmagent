package telemetry

import "time"

type Snapshot struct {
	Samples []ContainerSample
	Logs    []LogLine
}

type ContainerSample struct {
	ContainerID   string
	ContainerName string
	Image         string
	Status        string
	CPUPercent    float64
	MemUsageBytes uint64
	MemLimitBytes uint64
	MemPercent    float64
	NetRxBytes    uint64
	NetTxBytes    uint64
	NetRxPackets  uint64
	NetTxPackets  uint64
	BlkReadBytes  uint64
	BlkWriteBytes uint64
	RestartCount  int
	Timestamp     time.Time
	Labels        map[string]string
}

type LogLine struct {
	Timestamp     time.Time
	Severity      string
	Message       string
	ContainerName string
	ContainerID   string
	Stream        string
	TraceID       string
	SpanID        string
	Labels        map[string]string
}
