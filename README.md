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

## The private path (Project A — in progress)

A second ingest path where the collector **cannot** see any device's telemetry:
OTLP datapoints become Prio3 reports, secret-shared across two non-colluding
aggregators, and only a differentially-private aggregate is ever published.

```
agent ─OTLP/gRPC→ [ auth │ rate limit ] ─┬─▶ bounded queue ─▶ Kafka   (plaintext, existing)
                                         └─▶ encode ─▶ shard ─┬─▶ DAP leader ──┐
                                                              └─▶ DAP helper ──┤
                                                collector ◀── noised aggregate ─┘
```

| package | what |
|---|---|
| `internal/vdaf/field` | GF(2^64 − 2^32 + 1) arithmetic, polynomials |
| `internal/vdaf/xof` | seed expansion (TurboSHAKE128) |
| `internal/vdaf/prio3` | Prio3Count / Sum / Histogram + the FLP proof system |
| `internal/dap` | `draft-ietf-ppm-dap`: codec, HPKE, leader, helper, collector |
| `internal/dp` | Laplace / Gaussian / discrete Gaussian, RDP accountant, ε ledger |
| `internal/privacy` | OTLP → VDAF measurements, and the collector hook |

Implemented from the specs, with **no new module dependencies** — the whole
private path is Go standard library plus what the collector already used.

**Start here:** [docs/STUDY.md](docs/STUDY.md) maps all 71 exercises onto weeks
1–4 in dependency order. Then:

```bash
make test         # red — the failures ARE the task list, bottom-up
make test-field   # week 2 starts here; nothing above it can work first
make vectors      # the week-2 gate: the VDAF draft's own test vectors
make up-privacy   # Kafka + collector + DAP leader + helper
```

Design and reasoning: [docs/PRIVACY.md](docs/PRIVACY.md) ·
[docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) ·
[docs/BENCHMARKS.md](docs/BENCHMARKS.md)

### Keeping the two editions in sync

`scripts/strip-study.sh` generates the public (`main`) version of any file or
tree from this branch, by removing `// »` teaching comments, `EXERCISE-BEGIN …
EXERCISE-END` blocks, `«inline citations»` and study-only files. Write once
here; generate there.

```bash
scripts/strip-study.sh internal/dp/noise.go          # to stdout
scripts/strip-study.sh -o ../public-tree .           # whole tree, then gofmt -w
scripts/strip-study.sh --check internal/dp/noise.go  # nonzero if study content remains
```

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

Privacy-preserving measurement (Project A):

- VDAF — Prio3, Poplar1, test vectors in Appendix C:
  https://datatracker.ietf.org/doc/draft-irtf-cfrg-vdaf/
- DAP — the protocol the aggregators speak:
  https://datatracker.ietf.org/doc/draft-ietf-ppm-dap/
- DAP taskprov — task provisioning; co-authored by S. Wang (Apple) and
  C. Patton (Cloudflare):
  https://datatracker.ietf.org/doc/draft-ietf-ppm-dap-taskprov/
- Corrigan-Gibbs & Boneh, "Prio: Private, Robust, and Scalable Computation of
  Aggregate Statistics" (NSDI 2017) — the original, and very readable
- Near & Abuah, *Programming Differential Privacy* — https://programmingdp.com
- Canonne, Kamath & Steinke, "The Discrete Gaussian for Differential Privacy":
  https://arxiv.org/abs/2004.00010
- Mironov, "Rényi Differential Privacy": https://arxiv.org/abs/1702.07476
- RFC 9180 (HPKE) — also what Apple's Private Relay and OHTTP build on
- ISRG Divvi Up — reference implementations to read for shape, not to copy:
  https://github.com/divviup/libprio-rs and https://github.com/divviup/janus

The collector's original shape:

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

On the private path: Poplar1 / heavy hitters, homomorphic encryption, trusted
execution, and general MPC beyond additive secret sharing. Private *training*
across devices is Project B, a separate repository.
