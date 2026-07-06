# gpu-trace-collector

Multi-tenant telemetry ingestion service for
[gpu-trace](https://github.com/tiennvhust/gpu-trace): receives OTLP/gRPC
(metrics + logs) from agents, authenticates and rate-limits per tenant, and
produces the raw OTLP payloads to Kafka or Azure Event Hubs for downstream
stream processing.

```
agent ──OTLP/gRPC──▶ [ auth ▶ rate limit ▶ bounded queue ▶ kafka producer ] ──▶ telemetry.otlp
                       └────────────── stateless, horizontally scalable ─┘
```

## Design

- **Stateless.** Durability lives in Kafka; everything in-process is bounded
  and in-memory. Replicas scale horizontally behind a plain load balancer.
- **Backpressure over buffering.** The ingest queue is bounded; when full,
  requests are rejected fast with `RESOURCE_EXHAUSTED`, which OTLP SDK
  exporters retry with exponential backoff. Overload degrades into explicit,
  observable rejections — never into an OOM.
- **Per-tenant isolation.** Requests authenticate via an `x-api-key` gRPC
  header; rate limits are token buckets enforced in datapoints, so one noisy
  tenant cannot degrade the others.
- **OTLP passthrough.** Kafka record: key = tenant, value = unmodified OTLP
  protobuf, headers = `signal`, `encoding`. Downstream consumers share the
  public OTLP schema; the collector owns no schema of its own.
- **Delivery.** Producer runs idempotent with `acks=all` (no duplicates per
  partition on the produce side); end-to-end exactly-once is completed by
  consumers via idempotent writes.

## Layout

```
cmd/collector/        wiring + graceful shutdown
internal/config/      YAML config, env expansion, validation
internal/tenant/      auth interceptor + per-tenant token buckets
internal/server/      OTLP MetricsService/LogsService
internal/pipeline/    bounded queue + worker pool
internal/sink/        Kafka producer (franz-go)
internal/obs/         Prometheus self-metrics
configs/              collector.yaml (host) / collector.compose.yaml (compose)
deploy/               docker-compose: single-node KRaft Kafka + collector
```

## Quickstart

```bash
make up      # Kafka + collector via docker compose
# or natively against compose's Kafka:
make run
```

Point an agent at it (no agent changes; the OTel SDK carries the key as gRPC
metadata):

```bash
sudo OTEL_EXPORTER_OTLP_ENDPOINT=http://<collector-host>:4317 \
     OTEL_EXPORTER_OTLP_HEADERS="x-api-key=dev-key-1" \
     ./gpu-trace -pid <cuda-pid> -otel=grpc
```

Verify:

```bash
make consume                                        # records on telemetry.otlp
curl -s localhost:9464/metrics | grep collector_    # self-metrics
```

## Operations

- `:4317` OTLP/gRPC (agent-facing), `:9464` HTTP `/metrics` + `/healthz`,
  plus the standard gRPC health service for Kubernetes probes.
- Self-metrics: `collector_received_events_total{tenant,signal}`,
  `collector_rejected_events_total{tenant,reason}`,
  `collector_ingest_queue_depth`, `collector_produced_records_total{result}`.
- Shutdown drains front-to-back (stop gRPC → drain queue → flush producer),
  making rolling restarts lossless within the termination grace period.

## Azure Event Hubs

Event Hubs exposes a Kafka-protocol endpoint; switching is config-only:

```yaml
kafka:
  brokers: ["<namespace>.servicebus.windows.net:9093"]
  tls: true
  sasl:
    enabled: true
    username: "$ConnectionString"
    password: "${EVENTHUBS_CONN_STR}"
```

## Non-goals

Traces ingestion, payload transformation/enrichment, TLS termination, and
storage — these belong to other components of the pipeline (stream processor,
ClickHouse, query API).
