// Package pipeline implements the bounded queue between the gRPC handlers
// and the Kafka sink.
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
type Queue struct {
	ch   chan Item
	sink Sink
	m    *obs.Metrics
	wg     sync.WaitGroup
	policy string // "reject_new" | "drop_oldest"
}

// New creates the queue and starts `workers` drain goroutines.
func New(capacity, workers int, sink Sink, m *obs.Metrics, policy string) *Queue {
	q := &Queue{ch: make(chan Item, capacity), sink: sink, m: m, policy: policy}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	return q
}

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
func (q *Queue) Close() {
	close(q.ch)
	q.wg.Wait()
}

func (q *Queue) worker() {
	defer q.wg.Done()
	for it := range q.ch {
		if err := q.sink.Produce(context.Background(), it); err != nil {
			q.m.Produced.WithLabelValues("error").Inc()
			log.Printf("sink produce: %v", err)
		}
	}
}
