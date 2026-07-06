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
	wg   sync.WaitGroup
}

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
