package otlp

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	collectlogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"

	"github.com/argus/ecs-fargate-agent/internal/telemetry"
)

type Receiver struct {
	collectmetricspb.UnimplementedMetricsServiceServer

	addr    string
	service string
	logger  *slog.Logger
	server  *grpc.Server

	mu         sync.Mutex
	metrics    []telemetry.AppMetricSample
	network    []telemetry.NetworkCallSample
	logs       []telemetry.LogLine
	rollupSeen map[string]*telemetry.LogAggregate
}

type logsService struct {
	collectlogspb.UnimplementedLogsServiceServer
	receiver *Receiver
}

func NewReceiver(addr, service string, logger *slog.Logger) *Receiver {
	return &Receiver{
		addr:       addr,
		service:    service,
		logger:     logger,
		rollupSeen: make(map[string]*telemetry.LogAggregate),
	}
}

func (r *Receiver) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", r.addr)
	if err != nil {
		return err
	}
	r.server = grpc.NewServer()
	collectmetricspb.RegisterMetricsServiceServer(r.server, r)
	collectlogspb.RegisterLogsServiceServer(r.server, &logsService{receiver: r})

	go func() {
		<-ctx.Done()
		r.server.GracefulStop()
	}()
	go func() {
		if err := r.server.Serve(lis); err != nil {
			r.logger.Warn("probe local otlp receiver stopped", "error", err)
		}
	}()
	r.logger.Info("probe local otlp receiver started", "addr", r.addr)
	return nil
}

func (r *Receiver) Export(ctx context.Context, req *collectmetricspb.ExportMetricsServiceRequest) (*collectmetricspb.ExportMetricsServiceResponse, error) {
	var metrics []telemetry.AppMetricSample
	var network []telemetry.NetworkCallSample
	for _, rm := range req.GetResourceMetrics() {
		resourceAttrs := resourceAttrs(rm.GetResource())
		for _, sm := range rm.GetScopeMetrics() {
			for _, m := range sm.GetMetrics() {
				samples, calls := r.convertMetric(m, resourceAttrs)
				metrics = append(metrics, samples...)
				network = append(network, calls...)
			}
		}
	}
	r.mu.Lock()
	r.metrics = append(r.metrics, metrics...)
	r.network = append(r.network, network...)
	r.mu.Unlock()
	return &collectmetricspb.ExportMetricsServiceResponse{}, nil
}

func (s *logsService) Export(ctx context.Context, req *collectlogspb.ExportLogsServiceRequest) (*collectlogspb.ExportLogsServiceResponse, error) {
	r := s.receiver
	var lines []telemetry.LogLine
	var rollups []telemetry.LogAggregate
	for _, rl := range req.GetResourceLogs() {
		resourceAttrs := resourceAttrs(rl.GetResource())
		for _, sl := range rl.GetScopeLogs() {
			for _, rec := range sl.GetLogRecords() {
				line := r.convertLog(rec, resourceAttrs)
				lines = append(lines, line)
				rollups = append(rollups, aggregateFor(line))
			}
		}
	}
	r.mu.Lock()
	r.logs = append(r.logs, lines...)
	for _, rollup := range rollups {
		key := strings.Join([]string{rollup.ContainerName, rollup.Stream, rollup.Level, rollup.Fingerprint}, "\x00")
		if current, ok := r.rollupSeen[key]; ok {
			current.Count += rollup.Count
			if rollup.FirstSeen.Before(current.FirstSeen) {
				current.FirstSeen = rollup.FirstSeen
			}
			if rollup.LastSeen.After(current.LastSeen) {
				current.LastSeen = rollup.LastSeen
			}
			continue
		}
		cp := rollup
		r.rollupSeen[key] = &cp
	}
	r.mu.Unlock()
	return &collectlogspb.ExportLogsServiceResponse{}, nil
}

func (r *Receiver) Drain() (metrics []telemetry.AppMetricSample, network []telemetry.NetworkCallSample, logs []telemetry.LogLine, rollups []telemetry.LogAggregate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	metrics = r.metrics
	network = r.network
	logs = r.logs
	rollups = make([]telemetry.LogAggregate, 0, len(r.rollupSeen))
	for _, v := range r.rollupSeen {
		rollups = append(rollups, *v)
	}
	sort.Slice(rollups, func(i, j int) bool {
		return rollups[i].LastSeen.After(rollups[j].LastSeen)
	})
	r.metrics = nil
	r.network = nil
	r.logs = nil
	r.rollupSeen = make(map[string]*telemetry.LogAggregate)
	return metrics, network, logs, rollups
}

func (r *Receiver) convertMetric(m *metricspb.Metric, resourceAttrs map[string]string) ([]telemetry.AppMetricSample, []telemetry.NetworkCallSample) {
	switch data := m.GetData().(type) {
	case *metricspb.Metric_Gauge:
		return r.convertNumberPoints(m.GetName(), m.GetUnit(), resourceAttrs, data.Gauge.GetDataPoints(), false)
	case *metricspb.Metric_Sum:
		return r.convertNumberPoints(m.GetName(), m.GetUnit(), resourceAttrs, data.Sum.GetDataPoints(), true)
	case *metricspb.Metric_Histogram:
		return r.convertHistogram(m.GetName(), m.GetUnit(), resourceAttrs, data.Histogram.GetDataPoints())
	default:
		return nil, nil
	}
}

func (r *Receiver) convertNumberPoints(name, unit string, resourceAttrs map[string]string, points []*metricspb.NumberDataPoint, isCounter bool) ([]telemetry.AppMetricSample, []telemetry.NetworkCallSample) {
	metrics := make([]telemetry.AppMetricSample, 0, len(points))
	calls := make([]telemetry.NetworkCallSample, 0, len(points))
	for _, p := range points {
		labels := mergeAttrs(resourceAttrs, attrsToMap(p.GetAttributes()))
		value := numberValue(p)
		ts := pointTime(p.GetTimeUnixNano())
		metric := telemetry.AppMetricSample{
			Timestamp:     ts,
			ContainerName: containerName(labels, r.service),
			Namespace:     namespace(name),
			Name:          name,
			Value:         value,
			Unit:          unit,
			Labels:        labels,
		}
		metrics = append(metrics, metric)
		if isHTTPMetric(name, labels) {
			call := callFromLabels(metric.ContainerName, labels, ts)
			if isCounter && value > 0 {
				call.DurationMS = 0
			}
			calls = append(calls, call)
		}
	}
	return metrics, calls
}

func (r *Receiver) convertHistogram(name, unit string, resourceAttrs map[string]string, points []*metricspb.HistogramDataPoint) ([]telemetry.AppMetricSample, []telemetry.NetworkCallSample) {
	metrics := make([]telemetry.AppMetricSample, 0, len(points))
	calls := make([]telemetry.NetworkCallSample, 0, len(points))
	for _, p := range points {
		labels := mergeAttrs(resourceAttrs, attrsToMap(p.GetAttributes()))
		value := float64(0)
		if p.GetCount() > 0 {
			value = p.GetSum() / float64(p.GetCount())
		}
		ts := pointTime(p.GetTimeUnixNano())
		if strings.EqualFold(unit, "s") {
			value *= 1000
		}
		metric := telemetry.AppMetricSample{
			Timestamp:     ts,
			ContainerName: containerName(labels, r.service),
			Namespace:     namespace(name),
			Name:          name,
			Value:         value,
			Unit:          unit,
			Labels:        labels,
		}
		metrics = append(metrics, metric)
		if isHTTPMetric(name, labels) {
			call := callFromLabels(metric.ContainerName, labels, ts)
			call.DurationMS = value
			calls = append(calls, call)
		}
	}
	return metrics, calls
}

func (r *Receiver) convertLog(rec *logspb.LogRecord, resourceAttrs map[string]string) telemetry.LogLine {
	labels := mergeAttrs(resourceAttrs, attrsToMap(rec.GetAttributes()))
	return telemetry.LogLine{
		Timestamp:     pointTime(rec.GetTimeUnixNano()),
		Severity:      severity(rec),
		Message:       bodyString(rec.GetBody()),
		ContainerName: containerName(labels, r.service),
		Stream:        valueOr(labels["log.iostream"], "otel"),
		TraceID:       hex.EncodeToString(rec.GetTraceId()),
		SpanID:        hex.EncodeToString(rec.GetSpanId()),
		Labels:        labels,
	}
}

func attrsToMap(attrs []*commonpb.KeyValue) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		out[attr.GetKey()] = anyValueString(attr.GetValue())
	}
	return out
}

func resourceAttrs(resource *resourcepb.Resource) map[string]string {
	if resource == nil {
		return nil
	}
	return attrsToMap(resource.GetAttributes())
}

func mergeAttrs(a, b map[string]string) map[string]string {
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func anyValueString(v *commonpb.AnyValue) string {
	switch value := v.GetValue().(type) {
	case *commonpb.AnyValue_StringValue:
		return value.StringValue
	case *commonpb.AnyValue_IntValue:
		return strconv.FormatInt(value.IntValue, 10)
	case *commonpb.AnyValue_DoubleValue:
		return strconv.FormatFloat(value.DoubleValue, 'f', -1, 64)
	case *commonpb.AnyValue_BoolValue:
		return strconv.FormatBool(value.BoolValue)
	case *commonpb.AnyValue_BytesValue:
		return hex.EncodeToString(value.BytesValue)
	default:
		return ""
	}
}

func bodyString(v *commonpb.AnyValue) string {
	value := anyValueString(v)
	if value != "" {
		return value
	}
	return fmt.Sprint(v)
}

func numberValue(p *metricspb.NumberDataPoint) float64 {
	switch value := p.GetValue().(type) {
	case *metricspb.NumberDataPoint_AsDouble:
		return value.AsDouble
	case *metricspb.NumberDataPoint_AsInt:
		return float64(value.AsInt)
	default:
		return 0
	}
}

func pointTime(nanos uint64) time.Time {
	if nanos == 0 {
		return time.Now().UTC()
	}
	secs := int64(nanos / uint64(time.Second))
	ns := int64(nanos % uint64(time.Second))
	return time.Unix(secs, ns).UTC()
}

func namespace(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "app"
	}
	for _, sep := range []string{".", "_", "/"} {
		if i := strings.Index(name, sep); i > 0 {
			return strings.ToLower(name[:i])
		}
	}
	return "app"
}

func containerName(labels map[string]string, fallback string) string {
	for _, key := range []string{"container.name", "container_name", "service.name", "service"} {
		if v := strings.TrimSpace(labels[key]); v != "" {
			return v
		}
	}
	return fallback
}

func isHTTPMetric(name string, labels map[string]string) bool {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "http.") || strings.Contains(lower, "http_") {
		return true
	}
	_, hasStatus := labels["http.response.status_code"]
	if !hasStatus {
		_, hasStatus = labels["http.status_code"]
	}
	_, hasMethod := labels["http.request.method"]
	if !hasMethod {
		_, hasMethod = labels["http.method"]
	}
	return hasStatus || hasMethod
}

func callFromLabels(container string, labels map[string]string, ts time.Time) telemetry.NetworkCallSample {
	path := firstNonEmpty(labels, "url.path", "http.route", "http.target")
	if path == "" {
		path = pathFromURL(labels["url.full"])
	}
	return telemetry.NetworkCallSample{
		Timestamp:     ts,
		ContainerName: container,
		Direction:     direction(labels),
		Protocol:      firstNonEmpty(labels, "network.protocol.name", "url.scheme", "http.scheme"),
		Method:        firstNonEmpty(labels, "http.request.method", "http.method"),
		Peer:          firstNonEmpty(labels, "client.address", "net.peer.name", "net.peer.ip", "server.address"),
		Host:          firstNonEmpty(labels, "server.address", "http.host", "net.host.name"),
		Path:          path,
		StatusCode:    intValue(firstNonEmpty(labels, "http.response.status_code", "http.status_code")),
		Error:         firstNonEmpty(labels, "error.type", "exception.type"),
	}
}

func firstNonEmpty(labels map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(labels[key]); v != "" {
			return v
		}
	}
	return ""
}

func intValue(raw string) int {
	value, _ := strconv.Atoi(raw)
	return value
}

func pathFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.Path == "" {
		return "/"
	}
	return u.Path
}

func direction(labels map[string]string) string {
	if firstNonEmpty(labels, "http.route", "url.path", "http.target") != "" {
		return "inbound"
	}
	if firstNonEmpty(labels, "server.address", "net.peer.name") != "" {
		return "outbound"
	}
	return "unknown"
}

func severity(rec *logspb.LogRecord) string {
	if text := strings.TrimSpace(rec.GetSeverityText()); text != "" {
		return strings.ToLower(text)
	}
	switch {
	case rec.GetSeverityNumber() >= logspb.SeverityNumber_SEVERITY_NUMBER_ERROR:
		return "error"
	case rec.GetSeverityNumber() >= logspb.SeverityNumber_SEVERITY_NUMBER_WARN:
		return "warn"
	case rec.GetSeverityNumber() >= logspb.SeverityNumber_SEVERITY_NUMBER_INFO:
		return "info"
	default:
		return "debug"
	}
}

var volatileTokens = regexp.MustCompile(`(?i)\b[0-9a-f]{8,}\b|\b\d{2,}\b`)

func aggregateFor(line telemetry.LogLine) telemetry.LogAggregate {
	normalized := volatileTokens.ReplaceAllString(strings.ToLower(line.Message), "?")
	level := strings.ToLower(valueOr(line.Severity, "info"))
	sum := sha1.Sum([]byte(level + "|" + normalized))
	return telemetry.LogAggregate{
		ContainerName: line.ContainerName,
		Stream:        valueOr(line.Stream, "otel"),
		Level:         level,
		Fingerprint:   hex.EncodeToString(sum[:8]),
		Count:         1,
		SampleMessage: line.Message,
		FirstSeen:     line.Timestamp,
		LastSeen:      line.Timestamp,
	}
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

var _ collectmetricspb.MetricsServiceServer = (*Receiver)(nil)
var _ collectlogspb.LogsServiceServer = (*logsService)(nil)
var _ = (*resourcepb.Resource)(nil)
