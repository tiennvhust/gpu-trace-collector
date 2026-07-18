// Package sink produces telemetry records to Kafka (or any Kafka-protocol
// endpoint, e.g. Azure Event Hubs).
//
// » The Kafka record is the contract with the downstream stream processor:
// »   Key     = tenant name  → per-tenant ordering (same key, same partition)
// »   Value   = raw OTLP protobuf bytes (Export*ServiceRequest)
// »   Headers = signal (metrics|logs), encoding (otlp_proto_v1)
// » Keeping the value as unmodified OTLP means the processor can unmarshal
// » with the same public protos — no private schema to version.
// » Trade-off to know: keying by tenant risks hot partitions if one tenant
// » dominates. Alternatives (tenant+host composite key) relax ordering for
// » balance. https://developer.confluent.io/courses/architecture/get-started/
package sink

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"

	"github.com/tiennvhust/gpu-trace-collector/internal/config"
	"github.com/tiennvhust/gpu-trace-collector/internal/obs"
	"github.com/tiennvhust/gpu-trace-collector/internal/pipeline"
)

// Kafka wraps a franz-go producer client.
type Kafka struct {
	cl *kgo.Client
	m  *obs.Metrics
}

// New builds the client and verifies broker connectivity.
func New(cfg config.Kafka, m *obs.Metrics) (*Kafka, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.DefaultProduceTopic(cfg.Topic),

		// » acks=all + idempotent producing (franz-go's default) is the
		// » no-duplicates-per-partition produce-side guarantee. End to end
		// » you still only have at-least-once — exactly-once is finished by
		// » the CONSUMER (idempotent writes keyed on tenant+metric+window).
		// » https://www.confluent.io/blog/exactly-once-semantics-are-possible-heres-how-apache-kafka-does-it/
		kgo.RequiredAcks(kgo.AllISRAcks()),

		// » Linger trades a little latency for a lot of throughput: wait up
		// » to 25ms to fill produce batches. Tune this during load testing
		// » and record the effect in BENCHMARKS.md.
		kgo.ProducerLinger(25 * time.Millisecond),
		kgo.ProducerBatchCompression(kgo.SnappyCompression()),

		// » Second backpressure boundary: when this buffer is full, Produce
		// » blocks, workers stall, the pipeline queue fills, and the front
		// » door starts rejecting. Pressure propagates backwards — that
		// » chain reaction is the whole design.
		kgo.MaxBufferedRecords(1 << 16),
		kgo.ProduceRequestTimeout(10 * time.Second),
	}
	if cfg.TLS {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
	}
	if cfg.SASL.Enabled {
		// » Event Hubs: username "$ConnectionString", password = the
		// » namespace connection string. (Exercise for later: replace with
		// » OAUTHBEARER + workload identity when deploying to AKS.)
		opts = append(opts, kgo.SASL(plain.Auth{
			User: cfg.SASL.Username,
			Pass: cfg.SASL.Password,
		}.AsMechanism()))
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka client: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := cl.Ping(ctx); err != nil {
		cl.Close()
		return nil, fmt.Errorf("kafka ping %v: %w", cfg.Brokers, err)
	}
	return &Kafka{cl: cl, m: m}, nil
}

// EXERCISE-BEGIN
// ─── EXERCISE 4: dead-letter topic ──────────────────────────────────────────
// franz-go retries retriable produce errors internally, but some failures are
// terminal (record too large, topic authorization). Today those records are
// counted and dropped.
//
// Task: on terminal produce error, re-produce the record to a dead-letter
//       topic ("<topic>.dlq") with headers recording the error string and
//       original topic. Add a produced_total{result="dead_lettered"} counter.
// Think about: what must NOT happen if the DLQ produce ALSO fails? (Infinite
//       recursion is the classic bug here.)
// Pattern background: https://www.confluent.io/blog/error-handling-patterns-in-kafka/
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END

// Produce hands one item to the async producer.
//
// » Async with a completion callback: workers keep feeding batches instead of
// » stalling one RPC-round-trip per record (sync produce would cap
// » throughput at workers/latency records per second).
func (k *Kafka) Produce(ctx context.Context, it pipeline.Item) error {
	rec := &kgo.Record{
		Key:   []byte(it.Tenant),
		Value: it.Payload,
		Headers: []kgo.RecordHeader{
			{Key: "signal", Value: []byte(it.Signal)},
			{Key: "encoding", Value: []byte("otlp_proto_v1")},
		},
	}
	k.cl.Produce(ctx, rec, func(_ *kgo.Record, err error) {
		if err != nil {
			k.m.Produced.WithLabelValues("error").Inc()
			return
		}
		k.m.Produced.WithLabelValues("ok").Inc()
	})
	return nil
}

// Close flushes buffered records, then closes the client.
func (k *Kafka) Close(ctx context.Context) error {
	// » Flush before Close or the tail of the buffer is silently lost —
	// » the same bug class as the agent's tel.Shutdown() flush.
	if err := k.cl.Flush(ctx); err != nil {
		k.cl.Close()
		return fmt.Errorf("kafka flush: %w", err)
	}
	k.cl.Close()
	return nil
}
