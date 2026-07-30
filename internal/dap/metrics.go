package dap

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// » SAME RULE AS internal/obs: the pipeline must observe itself, because when it
// » breaks it takes the evidence with it. But a private aggregation pipeline adds
// » a rule of its own, and it is the more important one:
// »
// »   A METRIC ABOUT THE PRIVATE PATH MUST NOT LEAK WHAT THE PRIVATE PATH
// »   PROTECTS.
// »
// » It is very easy to build a beautiful DAP deployment and then defeat it with a
// » dashboard. Concretely, never label a series with:
// »   - a report ID (unbounded cardinality AND a per-report identifier, so your
// »     monitoring system becomes the report database you deliberately avoided
// »     keeping);
// »   - a client IP or device identifier;
// »   - any measurement value, or any bucket index. A histogram of "which bucket
// »     each report landed in" is exactly the data the whole protocol exists to
// »     hide, sitting in Prometheus in the clear.
// »
// » Label with task ID, role, and a bounded reason — nothing that varies per
// » device or per measurement. Then add a test that asserts the label sets are
// » exactly what you expect, so nobody adds `bucket` to a label list six months
// » from now. A cardinality guard and a privacy guard turn out to be the same
// » assertion, which is a satisfying result.
// »
// » Put this rule in THREAT_MODEL.md as its own actor row: "the monitoring system"
// » is a party that sees data, and it usually has weaker access controls than the
// » aggregator itself.

// Metrics holds the private path's self-metrics.
type Metrics struct {
	// ReportsReceived counts uploads accepted, by task.
	ReportsReceived *prometheus.CounterVec

	// ReportsRejected counts uploads refused, by task and reason. The reason
	// label is bounded: replay, expired, too_early, invalid_message,
	// unknown_task, queue_full.
	ReportsRejected *prometheus.CounterVec

	// ReportsVerified counts reports that passed or failed the VDAF check, by
	// task and result. A rising failure rate means a client bug or an attack —
	// two very different pages, so make sure the reason distinguishes them.
	ReportsVerified *prometheus.CounterVec

	// AggregationJobs counts jobs by task and outcome.
	AggregationJobs *prometheus.CounterVec

	// AggregationJobDuration is the cross-organisation round-trip latency. The
	// most important operational series here: it is the thing you do not control.
	AggregationJobDuration *prometheus.HistogramVec

	// HelperErrors counts helper failures by problem type, so a cross-org
	// incident is diagnosable from your own dashboard.
	HelperErrors *prometheus.CounterVec

	// Collections counts collections by task and outcome, including the batch-rule
	// rejections. A spike in batch_overlap is a collector misbehaving — or
	// attacking.
	Collections *prometheus.CounterVec

	// EpsilonSpent and EpsilonLimit expose the privacy ledger. Alert at 80%: an
	// exhausted budget means metrics silently stop updating, which is an outage
	// that looks like "the data went flat".
	EpsilonSpent *prometheus.GaugeVec
	EpsilonLimit *prometheus.GaugeVec

	reg *prometheus.Registry
}

// NewMetrics builds the registry and registers all collectors.
//
// TODO(week3): implement.
func NewMetrics() *Metrics {
	// EXERCISE-BEGIN
	// ─── EXERCISE 58: the private path's golden signals ─────────────────────
	// Task: build the series above, following internal/obs/metrics.go exactly —
	//       a private registry, explicit registration, the Go and process
	//       collectors.
	//
	// Then add the gauges that are NOT counters, because they are the ones that
	// tell you the system is healthy rather than merely busy:
	//   dap_pending_reports          — the aggregation queue depth. Rising means
	//                                  the helper cannot keep up with your ingest.
	//   dap_antireplay_set_size      — see store.go EXERCISE 42. This one OOMs you.
	//   dap_batch_reports{task}      — current batch fill against MinBatchSize, so
	//                                  you can see whether a task will ever be
	//                                  collectable. A task that never reaches its
	//                                  minimum produces no data at all and no
	//                                  errors — the worst kind of failure, and one
	//                                  only this gauge reveals.
	//
	// Then write the alerting rules, and be specific about what each one means:
	//   - epsilon_spent / epsilon_limit > 0.8               → budget nearly gone
	//   - rate(reports_verified{result="fail"}) > baseline   → client bug or attack
	//   - histogram_quantile(0.99, aggregation_job_duration) → helper degraded
	//   - dap_batch_reports < MinBatchSize for > 2 intervals → task producing
	//                                                          nothing, silently
	//
	// That last one is the alert that distinguishes someone who has operated a
	// system from someone who has built one.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	m := &Metrics{reg: prometheus.NewRegistry()}
	m.reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return m
}

// Handler returns the /metrics HTTP handler.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

// RegisterGauge wires a live value as a gauge, as internal/obs does for the
// ingest queue depth.
func (m *Metrics) RegisterGauge(name, help string, f func() float64) {
	m.reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, f))
}
