<!-- STUDY-ONLY-FILE -->
# Project A study map — weeks 1–4

The 71 exercises in this scaffold, in the order to do them, mapped onto the
8-week plan. Teaching comments in the source are marked `// »`; tasks are in
`EXERCISE-BEGIN … EXERCISE-END` blocks.

`make test` is **red** until you implement things, and that is the point: the
failures are your task list in dependency order. Work bottom-up. A failure in
`field` makes everything above it fail for reasons that have nothing to do with
its own code.

## How to use this scaffold

1. **Read the package doc comment first.** Every package opens with why it exists
   and how it fits. Those comments are the shortest path into the material.
2. **Read the spec section next to the code.** Each exercise names the draft
   section. Do not implement from the comment alone — the comments are a guide to
   the spec, not a replacement, and the test vectors check the spec.
3. **One layer at a time, tests green before moving up.**
4. **Never edit a test vector to make a test pass.** If they disagree, you are
   wrong.
5. **Write the docs as you go.** Many exercises end with "write this in
   docs/PRIVACY.md" or "…THREAT_MODEL.md". Those documents are deliverables, and
   the answers change once you have measured something.

## Week 1 — differential privacy

**Goal:** derive per-bucket histogram error from ε, n and bucket count on a
whiteboard.

| # | where | what |
|---|---|---|
| 1 | `dp/noise.go` | Laplace mechanism + the float-sampling caveat |
| 2 | `dp/noise.go` | Gaussian, classical then analytic (Balle & Wang) |
| 3 | `dp/noise.go` | **discrete Gaussian** (Canonne–Kamath–Steinke) — the one that ships |
| 4 | `dp/query.go` | bounded-sum sensitivity, and which DP definition you chose |
| 5 | `dp/query.go` | per-bucket standard deviation → the utility check |
| 6 | `dp/accountant.go` | RDP for Gaussian |
| 8 | `dp/accountant.go` | RDP → (ε, δ) conversion |
| 9 | `dp/accountant.go` | persistent ε ledger + gauges |

EXERCISE 7 (subsampling amplification) is week 6 — it belongs to Project B.

Reading: Near & Abuah ch. 1–8; Dwork & Roth ch. 2–3; Apple's "Learning with
Privacy at Scale"; Canonne–Kamath–Steinke.

**Checkpoint:** `make test-dp` green.

## Week 2 — secret sharing and Prio3

**Goal:** the draft's test vectors pass. This is the week that matters most.

| # | where | what |
|---|---|---|
| 10–12 | `vdaf/field/field.go` | Add, Sub, Mul (Goldilocks reduction), Inv |
| 13–14 | `vdaf/field/poly.go` | Horner evaluation, Lagrange interpolation |
| 15 | `vdaf/xof/xof.go` | **XofTurboShake128** — decide Route A or B and move on |
| 16 | `vdaf/xof/xof.go` | uniform field elements by rejection sampling |
| 17 | `vdaf/xof/xof_test.go` | pin the XOF with a KAT **before** using it |
| 18 | `vdaf/prio3/prio3.go` | the aggregation step |
| 19 | `vdaf/prio3/prio3.go` | additive sharing + the seed trick |
| 20 | `vdaf/prio3/prio3.go` | joint randomness, and why it is joint |
| 22 | `vdaf/prio3/flp.go` | the gadget polynomial |
| 27–28 | `vdaf/prio3/count.go` | Prio3Count — **build this one first** |
| 24–26 | `vdaf/prio3/driver.go` | Shard, PrepInit, Unshard |
| 29–31 | `vdaf/prio3/histogram.go` | Prio3Histogram — the one this project needs |
| 32 | `vdaf/prio3/sum.go` | Prio3Sum, bit decomposition, the range trap |
| 23 | `vdaf/prio3/flp.go` | **the six negative tests** — do not skip these |
| 34–35 | `vdaf/prio3/vectors_test.go` | the vector harness, then your own round trip |

Order within the week: field → xof (+KAT) → Prio3Count end to end → vectors for
Count → then Histogram → then Sum. Do **not** write all three variants before
running a single vector.

Reading: VDAF draft §1–7 (skip the FLP internals on the first pass); the Prio
paper.

**Checkpoint:** `make vectors` green. Record the output in the README.

## Week 3 — DAP, the aggregators, integration

**Goal:** end-to-end — a client uploads, two aggregators verify and aggregate, a
collector gets a noised result.

| # | where | what |
|---|---|---|
| 36–38 | `dap/codec.go` | TLS-presentation codec, then **fuzz it** |
| 39–40 | `dap/messages.go` | report decoding, the batch checksum |
| 41 | `dap/hpke.go` | HPKE seal/open, RFC 9180 vectors first |
| 42–45 | `dap/store.go` | anti-replay, **the three batch rules**, durability |
| 46–47 | `dap/task.go` | task validation, derived task IDs (taskprov) |
| 48–50 | `dap/leader.go` | upload path, collection jobs, the aggregation driver |
| 51–52 | `dap/helper.go` | the helper's step, and **where it says no** |
| 53–54 | `dap/collector.go` | collection ordering, independent noise per element |
| 55–56 | `dap/client.go` | config caching, the upload path |
| 58 | `dap/metrics.go` | the private path's golden signals |
| 68–70 | `cmd/dap-*`, `cmd/prio-client` | the three binaries |
| 59 | `dap/e2e_test.go` | **the end-to-end test** |

Reading: DAP draft §4–7; the taskprov draft (the Apple-co-authored one).

**Checkpoint:** `make up-privacy` then `./bin/prio-client -n 2000 -collect`
printing a noised aggregate next to the true value.

## Week 4 — ship, operate, write up

| # | where | what |
|---|---|---|
| 60 | `privacy/privacy.go` | **which metrics go on which path** — do this first |
| 61–62 | `privacy/encoder.go` | bucketing, OTLP → measurements, the attributes trap |
| 63–64 | `privacy/path.go` | the hot-path hook, the upload worker |
| 65 | `privacy/path.go` | **wire it into the collector** (4 small edits) |
| 66–67 | `privacy/config.go` | defaults, and validation that refuses bad privacy |
| 71 | `deploy/docker-compose.privacy.yml` | make the deployment honest |
| 21 | `vdaf/prio3/prio3.go` | Poplar1: write the paragraph, do not build it |
| 33 | `vdaf/prio3/sum.go` | Prio3SumVec, if time (high value, low cost) |
| 57 | `dap/client.go` | **the C++ client + on-device measurements** |

Then the documents:

- `docs/PRIVACY.md` — every **TODO**. This is the §7 design doc.
- `docs/THREAT_MODEL.md` — every actor row, and the honest gaps section.
- `docs/BENCHMARKS.md` — every table. §6 is the differentiator.
- README — the private-path section, with the test-vector output.
- Blog post #1.

**Checkpoint:** a stranger can clone, `make up-privacy`, and see a noised
aggregate.

## Triage, if you fall behind

Drop in this order (from the plan): the Swift track, then Poplar1 (EXERCISE 21 —
write the paragraph instead), then the secure-aggregation masking in Project B,
then blog post #2.

**Never drop:** EXERCISE 34 (the test vectors) or EXERCISE 57 (on-device energy
measurements). Those two are the whole point — one is objective proof you can
implement a spec, the other is the thing nobody else in the pipeline has.

## Exercises by theme, if you would rather work that way

- **The credibility gates:** 34 (vectors), 17 + 41 (KATs before use), 38 (fuzz)
- **The privacy invariants, all in ordinary code:** 42 (anti-replay), 43 (batch
  rules), 52 (independent helper checks), 53 (spend before collect), 67
  (refuse bad config)
- **The bandwidth/energy story:** 19 (seed trick), 29 (bucket cost), 33 (SumVec),
  56 + 57 (client measurement)
- **The systems engineering:** 50 (aggregation driver), 45 (durability), 63–64
  (hot-path hook), 71 (deployment)
- **Questions to answer in writing, not code:** 21, 30, 40, 47, 53 Task B, 60
