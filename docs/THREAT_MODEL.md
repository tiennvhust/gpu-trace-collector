# Threat model — the private aggregation path

<!-- STUDY-ONLY-BEGIN -->
> **The most valuable document in Project A, and the one most likely to be read
> first.** A threat model that says "the aggregators cannot see the data" is
> marketing. One that says, per actor, exactly what is learned and exactly what
> breaks under collusion is engineering — and it is the artifact that shows you
> understand the system rather than the paper.
>
> Rules for filling it in:
>
> 1. **Be specific about what leaks, including the awkward parts.** The FLP
>    reveals one bit (valid/invalid). The leader learns report timing. Say so.
>    A reviewer who finds an omission you should have known about will discount
>    everything else in the document.
> 2. **Separate enforced from contractual.** "The aggregators do not collude" is
>    not enforced by any code you wrote. Say which properties are cryptographic,
>    which are operational, and which are legal — that distinction is the single
>    clearest signal of someone who has thought about a real deployment.
> 3. **Write it as you implement**, because several rows only become visible once
>    you have built the thing (the metadata in `ReportMetadata`, the monitoring
>    row, the restart rows).
<!-- STUDY-ONLY-END -->

## Scope

Covers the private path only: client sharding through to the published noised
aggregate. The plaintext OTLP path is unchanged and out of scope — it is
explicitly a trusted-collector design, appropriate inside a tenant's own
boundary.

## What each actor learns

| actor | learns | does not learn | notes |
|---|---|---|---|
| **Client / device** | its own measurement | anything about other devices | **TODO** |
| **Network observer** | that a device uploaded, when, how many bytes, to which host | any measurement value | **TODO:** report sizes are fixed per task, so size does not leak the value — confirm this in your implementation |
| **Leader** | report IDs, report timestamps, source IP, its own input shares (uniformly random), one bit per report (valid / invalid) | any measurement value, the aggregate | **TODO** |
| **Helper** | report IDs, its own input shares, one bit per report | measurement values, the aggregate, source IPs (it never sees clients) | **TODO** |
| **Collector** | the noised aggregate, the report count, the batch interval, ε spent | any individual contribution, beyond what ε permits | **TODO** |
| **Monitoring / Prometheus** | task IDs, batch fill, verification failure rates, ε ledger | **TODO** — nothing per-device or per-measurement, *if* the label rule in `internal/dap/metrics.go` holds | Usually has weaker access controls than the aggregators. It is an actor. |
| **Aggregator operator (human)** | everything their service sees; plus logs | | **TODO:** what is in your logs? Grep for it before answering |

## What breaks under collusion or compromise

| scenario | consequence | mitigation |
|---|---|---|
| **Leader + helper collude** | Total loss. Both input shares reconstruct every measurement exactly. | Not cryptographic. Different organisations, contractual, and the config check that refuses identical URLs (`internal/privacy`« EXERCISE 67»). **Say plainly that this is the assumption the whole design rests on.** |
| Leader compromised alone | Report metadata, one bit per report, its own share. Can attempt the differencing attack — refused by the helper's independent batch checks. | prio3 verification; helper's independent rules (dap« EXERCISE 52») |
| Helper compromised alone | Its own share, one bit per report. Can deny service by refusing batches. | **TODO** |
| Collector compromised | Every published aggregate, which is what it is entitled to. | ε budget bounds the total disclosure across all of them |
| **Verify key leaked to clients** | Robustness gone: a client can forge a proof that verifies and poison the aggregate arbitrarily. Privacy is unaffected. | Secret store, never in config or logs (prio3« EXERCISE 25») |
| **Aggregator HPKE private key leaked** | That aggregator's shares become readable by whoever holds it — so leaking *both* is equivalent to collusion. | Secret store; rotation with overlapping validity (dap `hpke.go`) |
| **Leader restart** | In-memory anti-replay set is lost → replays possible. Collected-batch record is lost → the no-overlap rule stops being enforced, re-enabling differencing. | **Both are privacy failures, not availability ones.** dap« EXERCISE 45» |
| Malicious client, single | Bounded by the FLP: it can only submit a valid measurement, or be rejected. | prio3 verification |
| **Malicious clients, many (Sybil)** | A large number of colluding clients can shift the aggregate arbitrarily *within* the valid range. The FLP does not help. | **TODO:** what do you actually do? Device attestation is the real answer and it is outside DAP. Say so. |
| Collector requests overlapping batches | Differencing attack on the aggregates. | No-overlap rule, enforced independently by both aggregators (dap« EXERCISE 43») |
| Collector re-queries one batch | Averages away the noise: k queries reduce it by √k. | max_batch_query_count = 1, plus the ε ledger |

## Properties, and how each is actually guaranteed

Classify honestly. This table is the part a reviewer will scrutinise.

| property | mechanism | enforced by |
|---|---|---|
| No single aggregator sees a measurement | additive secret sharing over GF(p) | cryptography |
| Malformed client inputs are rejected | fully linear proof, random challenge | cryptography (soundness ≈ 1 − d/p) |
| Input shares are readable only by their aggregator | HPKE, AAD-bound to task and metadata | cryptography |
| One device cannot be counted twice | anti-replay set on report ID | **code, and only while the process lives** — see restart row |
| Batches are large enough | minimum batch size, checked by both aggregators | code, in two places |
| Batches do not overlap | interval intersection check, both aggregators | code, in two places |
| Aggregates reveal little about one device | discrete Gaussian noise at ε | code + the parameter choice |
| Cumulative disclosure is bounded | ε ledger per task per period | **code, and only while the process lives** |
| **The aggregators do not collude** | — | **nothing technical. Organisational and contractual.** |
| Clients are real devices, not Sybils | — | **TODO** — attestation, outside this system |

## Known gaps

<!-- STUDY-ONLY-BEGIN -->
> Fill these in from what you actually built and left undone. A gaps section
> that names real limitations reads as confidence; one that is empty reads as
> either dishonesty or inattention, and a reviewer will find the gaps anyway.
> Naming them first is strictly better.
<!-- STUDY-ONLY-END -->

1. **In-memory state.** Anti-replay set, collected-batch record and ε ledger do
   not survive a restart. Consequences above. (dap« EXERCISE 45»)
2. **Metadata.** The leader learns which device reported and when. Mitigation is
   an oblivious relay (OHTTP, iCloud Private Relay), which is out of scope here.
3. **Timing of collector uploads** from `internal/privacy` reveals tenant
   activity volume to the leader even though values are hidden.
   (`internal/privacy`« EXERCISE 64»)
4. **TODO:** whatever else you found. There will be several.

## References

- `draft-ietf-ppm-dap` — security considerations section
- Corrigan-Gibbs & Boneh, *Prio* — §3 threat model
- Apple Private Cloud Compute security guide — for contrast: a different point
  on the same trust/utility curve
