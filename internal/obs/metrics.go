// Package obs exposes the collector's OWN metrics.
package obs

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all collector self-metrics.
type Metrics struct {
	Received *prometheus.CounterVec
	Rejected *prometheus.CounterVec
	Produced *prometheus.CounterVec
	reg      *prometheus.Registry
}

// New builds the registry and registers all collectors.
func New() *Metrics {
	m := &Metrics{
		Received: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_received_events_total",
			Help: "Datapoints/log records accepted, by tenant and signal.",
		}, []string{"tenant", "signal"}),
		Rejected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_rejected_events_total",
			Help: "Rejected requests, by tenant and reason.",
		}, []string{"tenant", "reason"}),
		Produced: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_produced_records_total",
			Help: "Kafka produce results.",
		}, []string{"result"}),
		reg: prometheus.NewRegistry(),
	}
	m.reg.MustRegister(m.Received, m.Rejected, m.Produced,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return m
}

// RegisterQueueDepth wires the queue's live depth as a gauge.
func (m *Metrics) RegisterQueueDepth(depth func() int) {
	m.reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "collector_ingest_queue_depth",
		Help: "Items currently buffered in the ingest queue.",
	}, func() float64 { return float64(depth()) }))
}

// Handler returns the /metrics HTTP handler.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}
