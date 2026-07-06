// Package sink produces telemetry records to Kafka (or any Kafka-protocol
// endpoint, e.g. Azure Event Hubs).
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

		kgo.RequiredAcks(kgo.AllISRAcks()),

		kgo.ProducerLinger(25 * time.Millisecond),
		kgo.ProducerBatchCompression(kgo.SnappyCompression()),

		kgo.MaxBufferedRecords(1 << 16),
		kgo.ProduceRequestTimeout(10 * time.Second),
	}
	if cfg.TLS {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
	}
	if cfg.SASL.Enabled {
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

// Produce hands one item to the async producer.
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
	if err := k.cl.Flush(ctx); err != nil {
		k.cl.Close()
		return fmt.Errorf("kafka flush: %w", err)
	}
	k.cl.Close()
	return nil
}
