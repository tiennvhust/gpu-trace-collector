package privacy

import (
	"context"
	"sync"

	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"

	"github.com/tiennvhust/gpu-trace-collector/internal/dap"
)

// Path is the private ingest path: it takes OTLP requests off the collector's hot
// path, turns matching datapoints into DAP reports, and uploads them.
//
// The private path must never slow down or destabilise the plaintext path. So
// Ingest is non-blocking, the queue is bounded and drops rather than failing the
// OTLP request, and uploads are asynchronous — a dead DAP leader produces dropped
// measurements and a rising counter, never an error for the agent. The bounded
// queue and worker pool are the same shape as internal/pipeline, reused
// deliberately.
//
// » WHY THAT CONSTRAINT SHAPES EVERY DECISION HERE: the private path must never
// » slow down or destabilise the plaintext path. Prio3 sharding is orders of
// » magnitude more expensive than proto.Marshal, and it now sits inside an RPC
// » handler that already has a latency budget. So:
// »
// »   - Ingest is NON-BLOCKING. Enqueue and return; never shard on the gRPC
// »     goroutine.
// »   - The queue is BOUNDED. When full, drop and count — do NOT fail the OTLP
// »     request. The tenant's plaintext telemetry must survive the private path
// »     being broken or slow, because the private path is the new, less-proven
// »     component.
// »   - Uploads are asynchronous and independent. A dead DAP leader must not turn
// »     into OTLP errors for the agent.
// »
// » This is the same bounded-queue-plus-worker-pool shape as internal/pipeline,
// » reused deliberately: it is the pattern that already works in this codebase, the
// » reviewer already understands it, and the failure mode is already documented.
// » Reaching for the existing pattern rather than inventing one is itself the right
// » instinct to demonstrate.

// Path is safe for concurrent use.
type Path struct {
	tasks    []*taskBinding
	ch       chan job
	wg       sync.WaitGroup
	metrics  *dap.Metrics
	dropped  func()
	closeOne sync.Once
}

// taskBinding pairs a DAP task with the encoder that feeds it.
type taskBinding struct {
	task    *dap.Task
	encoder *Encoder
	client  *dap.Client
}

// job is one measurement awaiting upload.
type job struct {
	binding     *taskBinding
	measurement any
}

// New builds a private path from config. Returns nil when privacy is disabled, so
// callers can hold a nil *Path and let Ingest be a no-op.
//
// » A nil-safe type rather than an interface with a no-op implementation, because
// » it keeps the call site in internal/server free of a branch and makes "privacy
// » is off" the zero value. Small thing; it is the difference between the wiring
// » exercise below being three lines and being thirty.
//
// TODO(week4): implement.
func New(cfg Config, m *dap.Metrics) (*Path, error) { return nil, ErrTODO }

// Ingest extracts private measurements from an OTLP metrics request. It never
// blocks and never returns an error that should fail the caller's RPC.
//
// TODO(week4): implement.
func (p *Path) Ingest(req *colmetricspb.ExportMetricsServiceRequest) {
	if p == nil {
		return
	}
	// EXERCISE-BEGIN
	// ─── EXERCISE 63: the hot-path hook ─────────────────────────────────────
	// Task: walk the request's metrics, and for each metric matching a task's
	//       MetricSpec.MetricName, encode its datapoints and enqueue one job per
	//       measurement. Non-blocking send with a `default:` that counts a drop —
	//       exactly as internal/pipeline.Enqueue does.
	//
	// MEASURE THE OVERHEAD, because this is the number that decides whether the
	// design is acceptable:
	//   - benchmark the existing Export handler with the private path disabled;
	//   - benchmark it with the path enabled but matching nothing;
	//   - benchmark it matching one metric.
	// The middle number is the cost you impose on every tenant who does not use the
	// feature, and it must be near zero. If it is not, the name lookup is the
	// culprit — a map keyed on metric name, built once at startup, rather than a
	// linear scan over tasks per metric.
	//
	// Put all three numbers in docs/BENCHMARKS.md. "The private path costs
	// non-participating tenants X ns per request" is precisely the kind of claim
	// that makes a design review go quickly.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
}

// worker drains the queue, sharding and uploading.
//
// TODO(week4): implement.
func (p *Path) worker(ctx context.Context) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 64: the upload worker ─────────────────────────────────────
	// Task: for each job, call binding.client.Upload. On failure, retry with
	//       bounded exponential backoff and jitter, then drop and count.
	//
	// Sizing, and get this right before the load test rather than after: how many
	// workers? Sharding is CPU-bound (field arithmetic, XOF expansion) while
	// uploading is IO-bound, so one pool for both is wrong in both directions —
	// too few workers and the CPU idles during uploads, too many and sharding
	// thrashes. Either split the stages (shard pool sized to GOMAXPROCS, upload
	// pool sized to the leader's connection limit) or measure and accept a
	// compromise. Say which you did and why.
	//
	// Then a subtle one worth catching now: this worker sends measurements to an
	// EXTERNAL service. If the DAP leader is a third party, the timing and volume
	// of your uploads tells them about your tenants' activity even though the
	// values are hidden — the metadata leak from messages.go, now applied to your
	// own deployment rather than a device. Consider padding to a fixed rate, or
	// note it as accepted risk in THREAT_MODEL.md. Either is fine; noticing it is
	// the point.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
}

// Depth reports the current queue depth, for the gauge.
func (p *Path) Depth() int {
	if p == nil {
		return 0
	}
	return len(p.ch)
}

// Close drains the queue and stops the workers. Call it after the gRPC server has
// stopped accepting requests, in the same shutdown order the plaintext path uses.
//
// TODO(week4): implement — and see the shutdown-ordering note in
// cmd/collector/main.go: GracefulStop → queue.Close → privacy.Close → kafka.Close.
func (p *Path) Close() {
	if p == nil {
		return
	}
}

// EXERCISE-BEGIN
// ─── EXERCISE 65 (week 4): wire it into the collector ────────────────────────
// The private path is deliberately additive: nothing in the existing collector
// changed, so the plaintext path is exactly as proven as it was before. Wiring it
// in is your exercise, and it is four small edits. Doing it yourself is worth more
// than having it done for you — it is the moment the two halves of the project
// become one system.
//
// 1. internal/config/config.go — add the config block:
//
//	   Privacy privacy.Config `yaml:"privacy"`
//
//    ...and call its Validate() from Config.validate(). Note the import direction:
//    config importing privacy is fine; privacy must NOT import config, or you get
//    a cycle. If you find yourself wanting that, the Config type belongs in
//    internal/privacy (as it does here) and not in internal/config.
//
// 2. internal/server/otlp.go — add the field and one call. In MetricsService:
//
//	   type MetricsService struct {
//	       ...
//	       priv *privacy.Path   // nil when privacy is disabled
//	   }
//
//    and in Export, AFTER the rate-limit checks and AFTER the successful Enqueue:
//
//	   s.priv.Ingest(req)       // nil-safe, non-blocking
//
//    Think about WHERE. After rate limiting, so a tenant cannot use the private
//    path to bypass its quota. After Enqueue, so the plaintext path's latency is
//    unaffected by anything the private path does and a private-path failure can
//    never cost you a plaintext accept. Both orderings are deliberate; write the
//    reason in a comment, because the next reader will wonder.
//
// 3. cmd/collector/main.go — construct it and add it to the shutdown chain:
//
//	   priv, err := privacy.New(cfg.Privacy, dapMetrics)
//	   ...
//	   metrics.RegisterQueueDepth(...)  // add a private-path depth gauge too
//	   ...
//	   gs.GracefulStop()
//	   queue.Close()
//	   priv.Close()        // ← after the queue, before the sink
//	   kafka.Close(ctx)
//
//    Work out why priv.Close() goes there rather than first or last. (Which stages
//    can still be handed work by an earlier stage? The answer is the same argument
//    the existing comment in main.go makes about GracefulStop → queue → sink.)
//
// 4. Then PROVE the plaintext path is unaffected: run the existing collector with
//    privacy absent from the config file and confirm identical behaviour, then with
//    privacy configured and the DAP leader deliberately DOWN. In the second case
//    the agent must see no errors at all, `collector_received_events_total` must
//    keep climbing, and only the private-path drop counter should move.
//
//    That test — the new subsystem being completely down while the old one is
//    unaffected — is the difference between adding a feature and adding a risk. It
//    is also a very good thing to be able to describe in an interview when asked
//    how you ship changes to a production system.
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END
