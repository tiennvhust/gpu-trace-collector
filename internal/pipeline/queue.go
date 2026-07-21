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
	wg     sync.WaitGroup
	policy string // "reject_new" | "drop_oldest"
}

// dropOldestPolicy is the overload_policy value that evicts the head of the
// queue to make room for the newest item, instead of rejecting it.
const dropOldestPolicy = "drop_oldest"

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
func New(capacity, workers int, sink Sink, m *obs.Metrics, policy string) *Queue {
	q := &Queue{ch: make(chan Item, capacity), sink: sink, m: m, policy: policy}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	return q
}

// Enqueue adds an item, honoring the configured overload policy.
//
// » Non-blocking is deliberate: blocking here would smear queue-full latency
// » into every in-flight RPC and hide saturation from clients. Failing fast
// » keeps rejection cheap and observable. "Fail fast" discussion:
// » https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/
//
// » reject_new (default): the new item is refused; the caller sees
// » ResourceExhausted and retries with backoff. drop_oldest: the head of the
// » queue is evicted to make room, so the newest item always gets in — at
// » the cost of silently losing whatever was evicted (its producer was
// » already told "accepted").
func (q *Queue) Enqueue(it Item) error {
	select {
	case q.ch <- it:
		return nil
	default:
	}

	if q.policy != dropOldestPolicy {
		return ErrQueueFull
	}

	// » Keep evicting the head until the new item fits. Between our failed
	// » send above and the receive below, a worker may have already drained
	// » a slot — then the receive's default fires (nothing to evict) and we
	// » go straight to the send. Between our receive and our send, another
	// » producer may steal the slot we just freed — then the send's default
	// » fires and we loop back to evict again. Both races are harmless: they
	// » only change how many items we evict before our send succeeds, never
	// » whether it eventually does (the channel is bounded and only workers
	// » ever shrink it further, so we can't be racing something that grows
	// » the queue out from under us).
	for {
		select {
		case old := <-q.ch:
			// » The evicted item's producer already got an OK response —
			// » from its point of view the data vanished before Kafka, the
			// » same observable outcome as a produce error. Reuse the
			// » existing Rejected series (tenant + reason) rather than add a
			// » new metric family for it.
			q.m.Rejected.WithLabelValues(old.Tenant, "dropped_oldest").Inc()
		default:
		}

		select {
		case q.ch <- it:
			return nil
		default:
		}
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
