# gpu-trace-collector (learning edition)

Multi-tenant telemetry ingestion service: receives OTLP/gRPC from
[gpu-trace](https://github.com/tiennvhust/gpu-trace) agents, authenticates and
rate-limits per tenant, and produces the raw OTLP payloads to Kafka (or Azure
Event Hubs) for downstream stream processing.

```
agent ──OTLP/gRPC──▶ [ auth ▶ rate limit ▶ bounded queue ▶ kafka producer ] ──▶ telemetry.otlp
                       └────────────── stateless, horizontally scalable ─┘
```

This is the **learning edition**: teaching comments are marked `// »`, and
`EXERCISE` blocks describe follow-up tasks. The baseline compiles and works
with every exercise left undone.

## Design in one paragraph

The collector is deliberately stateless: durability lives in Kafka, and
everything in-process is bounded and in-memory. The bounded queue is the
backpressure boundary — when it fills, new requests are rejected with
`RESOURCE_EXHAUSTED` (the OTLP-standard retryable signal) instead of being
buffered toward an OOM. Rate limits are enforced per tenant, in datapoints
(not requests), so one noisy tenant cannot degrade the others. Payloads are
forwarded verbatim (key = tenant, value = raw OTLP protobuf) so the stream
processor shares the public OTLP schema with the agent — the collector never
becomes a schema owner.

## Layout

```
cmd/collector/        wiring + graceful shutdown ordering
internal/config/      YAML config, env expansion, validation
internal/tenant/      auth interceptor + per-tenant token buckets
internal/server/      OTLP MetricsService/LogsService (the hot path)
internal/pipeline/    bounded queue + worker pool (the backpressure boundary)
internal/sink/        franz-go Kafka producer (acks=all, idempotent)
internal/obs/         Prometheus self-metrics
configs/              collector.yaml (host) + collector.compose.yaml (compose)
deploy/               docker-compose: single-node KRaft Kafka + collector
```

## Run it

```bash
make up            # Kafka + collector (docker compose)
# or run natively against compose's Kafka:
make run           # uses configs/collector.yaml → localhost:29092
```

Point the agent at it — no agent code changes; the OTel SDK carries the API
key as gRPC metadata:

```bash
sudo OTEL_EXPORTER_OTLP_ENDPOINT=http://<collector-host>:4317 \
     OTEL_EXPORTER_OTLP_HEADERS="x-api-key=dev-key-1" \
     ./gpu-trace -pid <cuda-pid> -otel=grpc
```

Verify data is landing:

```bash
make consume                          # keys + headers of telemetry.otlp
curl -s localhost:9464/metrics | grep collector_   # self-metrics
```

You should see records keyed by tenant (`dev`) with headers
`signal=metrics|logs`, and `collector_received_events_total` climbing every
5s (the agent's export interval).

## Exercises

Each exercise is marked in the source with a banner comment containing hints
and references. Suggested order:

1. `internal/tenant` — constant-time key comparison + two-key rotation
2. `internal/tenant` — collector-wide (global) rate limiter
3. `internal/pipeline` — `drop_oldest` overload policy (and the essay question)
4. `internal/sink` — dead-letter topic for terminal produce errors
5. `internal/server` — OTLP `PartialSuccess` responses
6. `internal/server` — `RetryInfo` backoff hints on rejection
7. `cmd/collector` (stretch) — pprof + one measured hot-path optimization

Rule of the game: after each exercise, prove it with either a unit test or a
metric visible on `/metrics`, and write two sentences in this README about
the trade-off you chose. Those sentences are interview answers.

## Reading list (why this shape)

- Load shedding & bounded queues — AWS Builders' Library:
  https://aws.amazon.com/builders-library/using-load-shedding-to-avoid-overload/
- Retries, backoff, jitter:
  https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/
- OTLP protocol (responses, retryability, partial success):
  https://opentelemetry.io/docs/specs/otlp/
- OTel Collector architecture (the production-grade sibling of this design):
  https://opentelemetry.io/docs/collector/architecture/
- Kafka exactly-once, and why the consumer finishes the job:
  https://www.confluent.io/blog/exactly-once-semantics-are-possible-heres-how-apache-kafka-does-it/
- The Log (Kreps) — the conceptual foundation of this whole pipeline:
  https://engineering.linkedin.com/distributed-systems/log-what-every-software-engineer-should-know-about-real-time-datas-unifying
- Azure Event Hubs' Kafka endpoint (the zero-code-change cloud swap):
  https://learn.microsoft.com/en-us/azure/event-hubs/azure-event-hubs-apache-kafka-overview

## Non-goals (scope control)

Traces ingestion, payload transformation/enrichment, TLS termination (put a
proxy in front or do the TLS stretch exercise), and storage — those belong to
later stages of the project (stream processor, ClickHouse, query API).
