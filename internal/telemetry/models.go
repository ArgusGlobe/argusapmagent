package telemetry

import "time"

type Snapshot struct {
	Samples    []ContainerSample
	Logs       []LogLine
	AppMetrics []AppMetricSample
	Network    []NetworkCallSample
	LogRollups []LogAggregate
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

type AppMetricSample struct {
	Timestamp     time.Time
	ContainerName string
	Namespace     string
	Name          string
	Value         float64
	Unit          string
	Labels        map[string]string
}

type NetworkCallSample struct {
	Timestamp     time.Time
	ContainerName string
	Direction     string
	Protocol      string
	Method        string
	Peer          string
	Host          string
	Path          string
	StatusCode    int
	DurationMS    float64
	BytesIn       uint64
	BytesOut      uint64
	Error         string
}

type LogAggregate struct {
	ContainerName string
	Stream        string
	Level         string
	Fingerprint   string
	Count         uint64
	SampleMessage string
	FirstSeen     time.Time
	LastSeen      time.Time
}
