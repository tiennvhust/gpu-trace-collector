// Package pipeline implements the bounded queue between the gRPC handlers
// and the Kafka sink.
//
// » This queue IS the backpressure mechanism. Its two properties:
// »
// »   1. BOUNDED. An unbounded buffer converts overload into an OOM kill —
// »      the process "protects" callers right up until it dies. A bounded
// »      queue converts overload into fast, explicit rejections that clients
// »      retry with backoff. Read (short, excellent):
// »      https://aws.amazon.com/builders-library/using-load-shedding-to-avoid-overload/
// »
// »   2. DECOUPLING. The gRPC handler's latency stops depending on Kafka's
// »      latency: enqueue is O(1), so ingest p99 stays flat even while the
// »      sink rides out a broker hiccup (until the queue fills — by design).
// »
// » The OpenTelemetry Collector's equivalent knobs are the batch processor +
// » memory_limiter: https://opentelemetry.io/docs/collector/architecture/
package pipeline

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/tiennvhust/gpu-trace-collector/internal/obs"
)

// ErrQueueFull is returned by Enqueue when the bounded queue is at capacity.
// Handlers translate it into gRPC ResourceExhausted.
var ErrQueueFull = errors.New("ingest queue full")

// Item is one tenant-attributed OTLP payload headed for Kafka.
type Item struct {
	Tenant  string
	Signal  string // "metrics" | "logs"
	Payload []byte // serialized OTLP Export*ServiceRequest
}

// Sink consumes items; the Kafka producer implements it.
type Sink interface {
	Produce(ctx context.Context, it Item) error
	Close(ctx context.Context) error
}

// Queue is a bounded MPMC queue drained by a fixed worker pool.
//
// » A buffered channel is Go's native bounded MPMC queue — no locks, no
// » third-party library, and `len(ch)` gives us the depth gauge for free.
type Queue struct {
	ch   chan Item
	sink Sink
	m    *obs.Metrics
	wg   sync.WaitGroup
}

// EXERCISE-BEGIN
// ─── EXERCISE 3: an alternative overload policy ─────────────────────────────
// The baseline policy is reject-new: when full, the NEWEST data is refused
// and the client retries. The opposite policy is drop-oldest: evict the head
// to make room, on the theory that fresh telemetry is worth more than stale.
//
// Task: add a config field overload_policy: reject_new | drop_oldest and
//       implement drop_oldest in Enqueue (hint: a select that, on full,
//       does a non-blocking receive then retries the send — think carefully
//       about the race where a worker drains the slot between your receive
//       and your send, and why that race is harmless here).
// Then answer for your README: which policy is right for BILLING data?
//       For LIVE dashboards? (There is no single right answer — that's the
//       point. Kafka itself chose... look up log.retention vs quotas.)
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END

// New creates the queue and starts `workers` drain goroutines.
func New(capacity, workers int, sink Sink, m *obs.Metrics) *Queue {
	q := &Queue{ch: make(chan Item, capacity), sink: sink, m: m}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	return q
}

// Enqueue adds an item without blocking; it fails fast when full.
//
// » Non-blocking is deliberate: blocking here would smear queue-full latency
// » into every in-flight RPC and hide saturation from clients. Failing fast
// » keeps rejection cheap and observable. "Fail fast" discussion:
// » https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/
func (q *Queue) Enqueue(it Item) error {
	select {
	case q.ch <- it:
		return nil
	default:
		return ErrQueueFull
	}
}

// Depth reports the current number of buffered items (for the gauge).
func (q *Queue) Depth() int { return len(q.ch) }

// Close drains the queue: no new items are accepted (callers must stop the
// gRPC server first), buffered items are flushed, workers exit.
//
// » Shutdown ordering is a correctness property, not a nicety:
// »   GracefulStop() → Queue.Close() → sink.Close()
// » Any other order either drops buffered data or panics on send-on-closed-
// » channel. This is what makes rolling updates lossless.
func (q *Queue) Close() {
	close(q.ch)
	q.wg.Wait()
}

func (q *Queue) worker() {
	defer q.wg.Done()
	for it := range q.ch {
		// » context.Background(): the item's fate is now independent of the
		// » RPC that delivered it — the client was already told "accepted".
		if err := q.sink.Produce(context.Background(), it); err != nil {
			q.m.Produced.WithLabelValues("error").Inc()
			log.Printf("sink produce: %v", err)
		}
	}
}
