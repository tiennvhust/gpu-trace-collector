# Benchmarks — quantifying the privacy tax

<!-- STUDY-ONLY-BEGIN -->
> **A hiring manager on a billions-of-devices team will read this before the
> code.** "It's slower and bigger" is not an answer; "1024-bucket Prio3Histogram
> costs 8.2 KiB and 340 µs per report against 41 bytes and 2 µs for plaintext
> OTLP, and 11 µJ measured on a Cortex-A53" is.
>
> Fill every table from `make bench`. Empty cells are fine while you work; wrong
> numbers are not — record the machine, the Go version and the commit for each
> run, because you will compare across weeks and forget otherwise.
>
> The single most valuable row in this file is the last one in §5: measured energy
> on real hardware. Almost nobody applying for this role can produce it. You can.
<!-- STUDY-ONLY-END -->

Environment for all numbers below unless stated otherwise:

```
machine:   TODO (CPU, cores, RAM)
os/arch:   TODO
go:        TODO (go version)
commit:    TODO
command:   make bench
```

## 1. Field arithmetic

The foundation: everything above it is a multiple of these numbers.

| operation | ns/op | notes |
|---|---|---|
| `Add` | | |
| `Mul` (math/big reference) | |« EXERCISE 11 Task A» |
| `Mul` (Goldilocks reduction) | |« EXERCISE 11 Task B» |
| `Inv` | | ~100× `Mul`; check it is not in a per-report loop |

**Speedup from the Goldilocks reduction:** TODO×

**Fleet arithmetic:** at N field multiplications per Prio3Histogram report and
10⁹ devices reporting hourly, the difference between the two `Mul`
implementations is TODO CPU-hours/day. *That* calculation is why the field was
chosen.

## 2. Report size — the number that matters most on a device

Bytes on wire dominates energy cost. See `internal/dap/client.go`.

| VDAF | parameters | plaintext OTLP | Prio3 report | ratio |
|---|---|---|---|---|
| Prio3Count | — | | | |
| Prio3Sum | max 1000 (10 bits) | | | |
| Prio3Histogram | 16 buckets | | | |
| Prio3Histogram | 128 buckets | | | |
| Prio3Histogram | 1024 buckets | | | |
| Prio3SumVec | 50 × 10 bits | | |« EXERCISE 33» |

**Effect of the seed trick** (prio3« EXERCISE 19») at 1024 buckets:

| sharing | leader share | helper share | total |
|---|---|---|---|
| naive (both full vectors) | | | |
| seed-expanded helper share | | 16 B | |

**50 metrics: 50 × Prio3Sum vs 1 × Prio3SumVec:** TODO× smaller. The reason real
deployments use SumVec.

## 3. Client CPU and allocations

Allocations matter more than nanoseconds on a battery: allocation means GC, GC
means the CPU stays awake, and an awake CPU keeps the radio awake.

| operation | ns/op | B/op | allocs/op |
|---|---|---|---|
| `Shard` Prio3Count | | | |
| `Shard` Prio3Histogram (16) | | | |
| `Shard` Prio3Histogram (1024) | | | |
| HPKE `Seal` | | | |
| plaintext `proto.Marshal` (for scale) | | | |

**Where the time goes** (`-profile`, flame graph): TODO. Expect the XOF and field
multiplication to dominate — which is what« EXERCISE 11 Task B» is for.

## 4. Collector overhead — the cost imposed on tenants who do not use this

The private path must be nearly free for everyone else, or it cannot ship.

| configuration | ns/op per OTLP request | delta |
|---|---|---|
| private path disabled | | baseline |
| enabled, matching no metric | | **must be ≈ 0** |
| enabled, matching one metric | | |

If the middle row is not near zero, the metric-name lookup is a linear scan.
Make it a map built at startup. (`internal/privacy`« EXERCISE 63»)

## 5. End-to-end throughput and the cross-org link

| path | reports/s | p50 | p99 | notes |
|---|---|---|---|---|
| plaintext OTLP → Kafka | | | | existing collector |
| private, localhost helper | | | | |
| private, +50 ms to helper | | | | `netem`,« EXERCISE 71 Task C» |
| private, +200 ms to helper | | | | |

Throughput against helper RTT is a more interesting chart than raw localhost
throughput, and it is the one that shows the aggregation driver's concurrency was
sized deliberately«, EXERCISE 50».

**Aggregation batch size sweep** — reports per job vs throughput and p99 latency:
TODO. There is an optimum; find it.

## 6. Measured on real hardware

<!-- STUDY-ONLY-BEGIN -->
> **This is the differentiator.** Everything above is a laptop benchmark that any
> candidate could produce. This table needs a device, a build for it, and a
> current probe — and it is the intersection the whole 8-week plan is built
> around: privacy infrastructure measured under a real power budget.
>
> Do not drop this if you fall behind. Per the plan, the two things never to drop
> are the Prio3 test vectors and the on-device energy measurements.
<!-- STUDY-ONLY-END -->

| platform | language | report bytes | CPU per report | energy per report |
|---|---|---|---|---|
| x86-64 laptop | Go | | | n/a |
| ARM Cortex-A (phone/SBC) | C++ | | | |
| Cortex-M / MAX78002-class | C++ | | | |
| macOS | Swift | | | n/a |

Method: TODO — how you measured energy (current probe and shunt value, Battery
Historian, on-board fuel gauge), and over how many reports you averaged.

**The number that answers "can this ship on a battery":** at TODO µJ per report
and one report per hour, the private path costs TODO % of a TODO mWh daily
budget.

## 7. Reproducing

```bash
make bench                                   # everything below §5
make test-field -- -bench=Mul -benchmem      # §1
go test -bench=Shard -benchmem ./internal/vdaf/prio3   # §3
make up-privacy && ./bin/prio-client -n 10000 -mode histogram   # §5
./bin/prio-client -n 10000 -mode histogram -plaintext           # §5 baseline
```
