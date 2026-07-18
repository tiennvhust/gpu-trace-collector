// Package obs exposes the collector's OWN metrics.
//
// » Rule one of building telemetry systems: the pipeline must observe
// » itself, because when it breaks it takes the evidence with it. These four
// » series answer the four operational questions:
// »   received_events_total  — is data flowing, and from whom?
// »   rejected_events_total  — are we shedding load, and WHY?
// »   ingest_queue_depth     — how close to saturation are we?
// »   produced_records_total — is data leaving toward Kafka?
// » Together they are the pipeline's "golden signals":
// » https://sre.google/sre-book/monitoring-distributed-systems/
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
//
// » A private registry (not prometheus.DefaultRegisterer) keeps the exposed
// » series an explicit, reviewable list — a habit that matters at fleet
// » scale where accidental cardinality is a real outage class:
// » https://prometheus.io/docs/practices/naming/
func New() *Metrics {
	m := &Metrics{
		Received: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "collector_received_events_total",
			Help: "Datapoints/log records accepted, by tenant and signal.",
		}, []string{"tenant", "signal"}),
		Rejected: prometheus.NewCounterVec(prometheus.CounterOpts{
			// » The reason label is the money label: rate_limit vs queue_full
			// » vs unauthenticated are three different pages to be woken by.
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
