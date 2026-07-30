# The private aggregation path

<!-- STUDY-ONLY-BEGIN -->
> **This document is a deliverable, not documentation.** It is the §7 design doc
> from the 8-week plan — *"How I'd collect evaluation telemetry for an on-device
> generative model across 10⁹ devices without seeing any user's output"* — with
> the nouns changed to GPU telemetry. Every `**TODO:**` below is a question you
> answer from your own implementation. Filling them in *in writing* is what
> separates you from candidates improvising in the room.
>
> Write it as you build, not at the end. The answers change once you have measured
> something.
<!-- STUDY-ONLY-END -->

The collector's original path sees every tenant's raw telemetry in the clear.
This document describes the path where it cannot.

```
                        ┌──────────────── plaintext path (existing) ─────────────┐
agent ──OTLP/gRPC──▶ [ auth ▶ rate limit ] ─┬─▶ bounded queue ─▶ Kafka ─▶ per-device rows
                                            │
                                            └─▶ encode ─▶ shard ─┬─▶ DAP leader ──┐
                                                                 │                │ aggregate
                                                                 └─▶ DAP helper ──┤ shares
                                                                                  ▼
                                                    collector ◀── noised aggregate only
```

## Why both paths exist

Per-device telemetry inside a tenant's own trust boundary is not a privacy
problem — attribution *is* the product there. The private path exists for the
data that crosses a boundary: fleet-wide distributions, cross-tenant benchmarks,
vendor-facing statistics. "Is my 40% GPU utilisation good?" cannot be answered
without everyone else's numbers, and nobody will hand those over in the clear.

**TODO:** the metric-by-metric decision from `internal/privacy/privacy.go`
«EXERCISE 60 — »which of SM utilisation, memory occupancy, kernel duration,
kernel launch count, CUDA errors, process name and hostname goes on which path,
and why.

## The two mechanisms, and why one is not enough

| | hides | does not hide |
|---|---|---|
| Secret sharing (Prio3) | each individual report, from each aggregator | the exact published aggregate |
| Differential privacy | what any aggregate reveals about one device | anything, from a party holding raw data |

Secret sharing alone publishes an exact sum, and exact sums over two overlapping
batches subtract to one device's contribution. DP alone requires someone to hold
the raw values in order to noise them. The design needs both, plus batch
discipline — a minimum batch size, no overlapping batches, one collection per
batch — because DP bounds what a *sequence* of releases reveals only if the
sequence is the one you accounted for.

## Components

| package | role |
|---|---|
| `internal/vdaf/field` | GF(2^64 − 2^32 + 1) arithmetic and polynomials |
| `internal/vdaf/xof` | seed expansion (TurboSHAKE128) |
| `internal/vdaf/prio3` | Prio3Count / Sum / Histogram, the FLP proof system |
| `internal/dap` | the protocol: codec, messages, HPKE, leader, helper, collector |
| `internal/dp` | noise mechanisms, sensitivity, the RDP accountant and ε ledger |
| `internal/privacy` | OTLP datapoints → VDAF measurements, and the collector hook |

Specifications implemented:

- `draft-irtf-cfrg-vdaf` — Prio3 (§7), fields (§6), XOFs (§6.2)
- `draft-ietf-ppm-dap` — the protocol
- `draft-ietf-ppm-dap-taskprov` — derived task IDs
- RFC 9180 — HPKE
- RFC 8446 §3 — the TLS presentation language used for DAP messages

**TODO:** the draft revision each was implemented against, and the test-vector
result.

## Parameter choices

Every number below is a trade-off someone has to defend.

| parameter | value | reasoning |
|---|---|---|
| VDAF | | **TODO** |
| buckets | | **TODO** — and what bucket count does to relative error |
| bucketing | | **TODO** — linear vs log, per metric |
| chunk length | | **TODO** — the proof-size/gadget-degree trade (prio3« EXERCISE 29») |
| ε per collection | | **TODO** |
| δ | | **TODO** — and why it must be ≪ 1/n |
| ε budget per period | | **TODO** — how many collections that funds |
| minimum batch size | | **TODO** — derived from the per-bucket noise, not guessed |
| time precision | | **TODO** — the privacy/timeliness trade |
| report expiry | | **TODO** — clock skew vs pinned storage (dap« EXERCISE 39») |
| max query count | | **TODO** — why 1 |

## Design decisions to record

Each of these is a question the exercises raise and the code cannot answer on its
own.

1. **Which DP definition** — add/remove a row (unbounded DP) or replace a row
   (bounded DP)? The sensitivity differs by 2×. (`internal/dp`« EXERCISE 4»)
2. **Why noise at the collector**, not inside `Unshard`, and how the single path
   from exact aggregate to published value is enforced. (prio3« EXERCISE 26»)
3. **Negative noisy counts** — published signed, or clamped? Why clamping biases
   downstream statistics and why post-processing is free. (prio3« EXERCISE 30»)
4. **Per-device contribution bound** when one host has eight GPUs: aggregate
   locally, inflate the sensitivity, or sample one GPU per batch?
   (`internal/privacy`« EXERCISE 62»)
5. **Budget refresh** — resetting ε per period is sound only under an assumption
   about a user's data appearing in one period. State the assumption.
   (`internal/dp`« EXERCISE 9»)
6. **No refund on a failed collection**, because a refund is an oracle: retry
   until the noise is favourable. (dap« EXERCISE 53»)
7. **Whether ε belongs in the derived task ID.** (dap« EXERCISE 47»)
8. **Whether the report count is published**, and if so, that it needs its own
   noise and its own share of the budget. (`internal/dap/collector.go`)
9. **What the checksum mismatch policy is**, and what a malicious leader gains
   from causing one. (dap« EXERCISE 40»)
10. **Small tenants.** `MinBatchSize` counts reports, not tenants. A tenant with
    three GPUs is not anonymous in a fleet aggregate however good the crypto is.
    **TODO:** what you do about it. (The answer is not cryptographic.)

## Operating it

| signal | meaning | action |
|---|---|---|
| `dap_epsilon_spent / dap_epsilon_limit > 0.8` | budget nearly gone | plan the next period, or reduce collections |
| `dap_batch_reports < min_batch_size` for > 2 intervals | the task produces **nothing**, silently | lower the minimum, lengthen the interval, or accept no data |
| `rate(dap_reports_verified{result="fail"})` above baseline | client bug, or an attack | check for a recent client rollout first |
| `dap_antireplay_set_size` growing without bound | eviction is not working | this OOMs the aggregator |
| `dap_aggregation_job_duration` p99 rising | the helper is degraded | it is not your service; you have a runbook, not a fix |

The failure mode to fear is not an outage. It is a task that quietly produces
noise, or no data, while every dashboard stays green — which is why the
`min_batch_size` gauge and the utility check at startup
(`internal/privacy`« EXERCISE 67») exist.

## What this does not protect

See [THREAT_MODEL.md](THREAT_MODEL.md) for the actor-by-actor account. In
summary: the leader learns *that* a device reported and *when*; the network sees
the connection; a colluding leader and helper learn everything; and the
monitoring system is an actor with weaker access controls than the aggregators.

## Non-goals

Poplar1 / heavy hitters (see `internal/vdaf/prio3`« EXERCISE 21» for the reasoning
about why not), homomorphic encryption, trusted execution, general MPC beyond
additive secret sharing, and private *training* — that is Project B.
