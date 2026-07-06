// Package server implements the OTLP/gRPC receiver: the MetricsService and
// LogsService Export RPCs that the agent's SDK exporters call.
package server

import (
	"context"
	"time"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/tiennvhust/gpu-trace-collector/internal/obs"
	"github.com/tiennvhust/gpu-trace-collector/internal/pipeline"
	"github.com/tiennvhust/gpu-trace-collector/internal/tenant"
)

// MetricsService implements the OTLP metrics Export RPC.
type MetricsService struct {
	colmetricspb.UnimplementedMetricsServiceServer
	q *pipeline.Queue
	m *obs.Metrics
}

// LogsService implements the OTLP logs Export RPC.
type LogsService struct {
	collogspb.UnimplementedLogsServiceServer
	q *pipeline.Queue
	m *obs.Metrics
}

// Register attaches both services to the gRPC server.
func Register(s *grpc.Server, q *pipeline.Queue, m *obs.Metrics) {
	colmetricspb.RegisterMetricsServiceServer(s, &MetricsService{q: q, m: m})
	collogspb.RegisterLogsServiceServer(s, &LogsService{q: q, m: m})
}

// Export handles one metrics export request:
// count → rate-limit → serialize → enqueue.
func (s *MetricsService) Export(ctx context.Context,
	req *colmetricspb.ExportMetricsServiceRequest,
) (*colmetricspb.ExportMetricsServiceResponse, error) {

	t := tenant.FromContext(ctx)
	if t == nil {
		return nil, status.Error(codes.Internal, "no tenant in context")
	}

	n := countDataPoints(req)
	if n == 0 {
		return &colmetricspb.ExportMetricsServiceResponse{}, nil
	}

	if !t.Limiter.AllowN(time.Now(), n) {
		s.m.Rejected.WithLabelValues(t.Name, "rate_limit").Inc()
		return nil, status.Error(codes.ResourceExhausted,
			"per-tenant rate limit exceeded, retry with backoff")
	}

	payload, err := proto.Marshal(req)
	if err != nil {
		return nil, status.Error(codes.Internal, "marshal: "+err.Error())
	}

	if err := s.q.Enqueue(pipeline.Item{Tenant: t.Name, Signal: "metrics", Payload: payload}); err != nil {
		s.m.Rejected.WithLabelValues(t.Name, "queue_full").Inc()
		return nil, status.Error(codes.ResourceExhausted,
			"ingest queue full, retry with backoff")
	}

	s.m.Received.WithLabelValues(t.Name, "metrics").Add(float64(n))
	return &colmetricspb.ExportMetricsServiceResponse{}, nil
}

// Export handles one logs export request (same pipeline as metrics).
func (s *LogsService) Export(ctx context.Context,
	req *collogspb.ExportLogsServiceRequest,
) (*collogspb.ExportLogsServiceResponse, error) {

	t := tenant.FromContext(ctx)
	if t == nil {
		return nil, status.Error(codes.Internal, "no tenant in context")
	}

	n := countLogRecords(req)
	if n == 0 {
		return &collogspb.ExportLogsServiceResponse{}, nil
	}
	if !t.Limiter.AllowN(time.Now(), n) {
		s.m.Rejected.WithLabelValues(t.Name, "rate_limit").Inc()
		return nil, status.Error(codes.ResourceExhausted,
			"per-tenant rate limit exceeded, retry with backoff")
	}
	payload, err := proto.Marshal(req)
	if err != nil {
		return nil, status.Error(codes.Internal, "marshal: "+err.Error())
	}
	if err := s.q.Enqueue(pipeline.Item{Tenant: t.Name, Signal: "logs", Payload: payload}); err != nil {
		s.m.Rejected.WithLabelValues(t.Name, "queue_full").Inc()
		return nil, status.Error(codes.ResourceExhausted,
			"ingest queue full, retry with backoff")
	}

	s.m.Received.WithLabelValues(t.Name, "logs").Add(float64(n))
	return &collogspb.ExportLogsServiceResponse{}, nil
}

// countDataPoints walks a metrics request and counts individual datapoints —
// the unit for rate limiting and the received_events metric.
func countDataPoints(req *colmetricspb.ExportMetricsServiceRequest) int {
	n := 0
	for _, rm := range req.GetResourceMetrics() {
		for _, sm := range rm.GetScopeMetrics() {
			for _, mt := range sm.GetMetrics() {
				switch d := mt.GetData().(type) {
				case *metricspb.Metric_Gauge:
					n += len(d.Gauge.GetDataPoints())
				case *metricspb.Metric_Sum:
					n += len(d.Sum.GetDataPoints())
				case *metricspb.Metric_Histogram:
					n += len(d.Histogram.GetDataPoints())
				case *metricspb.Metric_ExponentialHistogram:
					n += len(d.ExponentialHistogram.GetDataPoints())
				case *metricspb.Metric_Summary:
					n += len(d.Summary.GetDataPoints())
				}
			}
		}
	}
	return n
}

// countLogRecords walks a logs request and counts log records.
func countLogRecords(req *collogspb.ExportLogsServiceRequest) int {
	n := 0
	for _, rl := range req.GetResourceLogs() {
		for _, sl := range rl.GetScopeLogs() {
			n += len(sl.GetLogRecords())
		}
	}
	return n
}
