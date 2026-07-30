package dap

import (
	"context"
	"net/http"
	"time"
)

// » THE CLIENT IS THE PART THAT RUNS ON THE DEVICE, so every decision here is a
// » battery decision. This is where your six years of milliwatt budgets is worth
// » more than anyone else's knowledge of the crypto, and it is the differentiator
// » the plan is built around — so instrument it properly rather than treating it
// » as the easy end of the protocol.
// »
// » What actually costs energy, in order:
// »   1. THE RADIO, by a wide margin. Bytes on wire, and more importantly the
// »      number of times you wake the radio. One 8 KiB upload per hour is far
// »      cheaper than sixty 140-byte uploads per hour, because the tail energy of
// »      keeping the radio in a high-power state dominates the transmission
// »      itself.
// »   2. The HPKE seal — two of them per report, one per aggregator.
// »   3. Prio3 sharding: field arithmetic and XOF expansion.
// »   4. Allocation, indirectly, via GC — which keeps the CPU awake, which keeps
// »      everything else awake.
// »
// » So the operational advice that falls out is: BATCH AND COALESCE. Do not upload
// » on measurement; buffer locally, upload once per window, and align the upload
// » with something else that already wakes the radio. On iOS that is BGTaskScheduler
// » and the system's opportunistic scheduling; on Android, WorkManager with a
// » network constraint; on a bare MCU, whatever your existing duty cycle is. Say
// » this out loud in an interview — it is a device-engineering answer to a
// » privacy-infrastructure question, and almost nobody in the pipeline for this
// » role can give it.

// Client uploads reports to a DAP leader.
type Client struct {
	task       *Task
	leaderHPKE *HPKEConfig
	helperHPKE *HPKEConfig
	http       *http.Client
}

// NewClient constructs a client for one task, fetching both aggregators' HPKE
// configs.
//
// TODO(week3): implement.
func NewClient(ctx context.Context, task *Task, hc *http.Client) (*Client, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 55: config fetch and cache ────────────────────────────────
	// Fetch /hpke_config from both aggregators and pick a config from each.
	//
	// On a real device this must be cached persistently and refreshed lazily —
	// fetching two configs before every upload doubles the radio wake-ups to send
	// one report, which is the single worst thing you can do to a battery here.
	// Cache with the max-age from Cache-Control, and treat a fetch failure as
	// "keep using the cached config" rather than "drop the measurement": a
	// slightly stale key still works (that is what the overlapping validity in
	// hpke.go is for), whereas a dropped measurement is gone forever.
	//
	// Then think about the privacy angle, which is easy to miss: fetching the
	// config is itself a network request that reveals the device participates in
	// this task, and its timing correlates with the upload. Batch the fetch with
	// the upload, or fetch on a schedule unrelated to reporting.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// Upload shards a measurement and uploads the report.
//
// TODO(week3): implement.
func (c *Client) Upload(ctx context.Context, measurement any) error {
	// EXERCISE-BEGIN
	// ─── EXERCISE 56: the client's five steps, and what each costs ───────────
	//   1. NewReportID() and a rounded timestamp (TimePrecision).
	//   2. task.VDAF.Shard(measurement, nonce, rand).
	//   3. InputShareAAD, then Seal each input share to its aggregator's key.
	//   4. Encode the Report.
	//   5. POST to the leader with Content-Type: MediaTypeReport.
	//
	// Handle the responses properly, because the retry policy is a battery policy:
	//   400-class → the report is bad. DROP IT. Do not retry; a retry cannot
	//     succeed and burns radio time. Count it so you can see a bad rollout.
	//   503 with Retry-After → the leader is overloaded. Back off WITH JITTER, and
	//     respect Retry-After. This is the same signal the OTLP exporter handles on
	//     the plaintext path.
	//   Network error → retry with a bounded budget, then persist the report
	//     locally and try on the next scheduled wake-up. Do NOT retry in a tight
	//     loop on a battery.
	//
	// MEASURE, and this is the deliverable (docs/BENCHMARKS.md):
	//   - report bytes on wire, per variant and per parameter;
	//   - client CPU time per report (go test -bench);
	//   - allocations per report — the GC proxy for energy;
	//   - and if you have a device: actual energy per report. A phone with
	//     Battery Historian, or your MAX78002-class board with a current probe.
	//     One real measured number here is worth more than the whole rest of the
	//     table, because it is the number nobody else will have.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return ErrTODO
}

// BatchingClient buffers measurements and uploads them on a schedule, to keep the
// radio asleep.
//
// » The shape that actually ships. The naive client uploads on measurement; this
// » one accumulates and flushes on a timer or at a size threshold — the same
// » trade-off the OTel batch exporter makes on the plaintext path, for the same
// » reason, with battery rather than throughput as the payoff.
type BatchingClient struct {
	*Client
	FlushInterval time.Duration
	MaxBuffered   int
}

// EXERCISE-BEGIN
// ─── EXERCISE 57 (the one that makes Project A yours): a C++ client ──────────
// A Go client proves the protocol. A client in a language that runs on
// constrained devices proves you understand the deployment, and it is explicitly
// in the plan's deliverables ("a client in C++ ... proving the protocol works
// from a constrained device").
//
// Scope it tightly — a full Prio3 implementation in C++ is not the point:
//   - Field64 arithmetic and the XOF (or link libsodium/BoringSSL for the hash);
//   - Prio3Count and Prio3Histogram sharding only, no aggregator side;
//   - HPKE seal only, no open;
//   - a plain HTTP POST.
//
// That is a few hundred lines and it goes in a cpp/ subdirectory with a CMake
// build, which is your home turf. Then MEASURE IT on real hardware and put the
// comparison in docs/BENCHMARKS.md:
//
//   platform          | report bytes | CPU per report | energy per report
//   Go, x86-64        |              |                | n/a
//   C++, ARM Cortex-A |              |                |
//   C++, Cortex-M     |              |                |
//
// If you have a Mac, add a Swift client too (plan §2.4) — a small macOS CLI that
// submits a DAP report is enough to say "I have written Swift" truthfully and
// answer follow-ups. If you do not, say so plainly; the JD accepts embedded
// experience as the substitute and this table is the evidence.
//
// The last row of that table is the sentence your CV is built around: most
// candidates for this team know the crypto or know the device. This proves you
// measured the crypto ON the device.
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END
