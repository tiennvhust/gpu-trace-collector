package dap

import (
	"fmt"
	"sync"
	"time"

	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/prio3"
)

// » THIS FILE IS WHERE THE PRIVACY GUARANTEES ACTUALLY LIVE, and it contains no
// » cryptography at all. Anti-replay, minimum batch size and no-overlap are
// » ordinary data-structure discipline, and every one of them is load-bearing:
// » break any one and the crypto above it stops protecting anyone. If you take one
// » thing from Project A into an interview, make it this — the privacy
// » engineering was systems engineering.

// ErrReplay is returned when a report ID has already been seen.
var ErrReplay = fmt.Errorf("dap: report already seen")

// ErrBatchTooSmall is returned when a collection would cover fewer reports than
// the task's minimum batch size.
var ErrBatchTooSmall = fmt.Errorf("dap: batch smaller than the task minimum")

// ErrBatchOverlap is returned when a collection would reuse reports already
// included in a previous collection.
var ErrBatchOverlap = fmt.Errorf("dap: batch overlaps a previously collected batch")

// Store holds an aggregator's state for one task: the anti-replay set, the
// running aggregate share, and the record of which batches have been collected.
//
// The three rules enforced here — no replays, a minimum batch size, and no
// overlapping or repeated collections — are what make the DP accounting true
// rather than nominal. None of them involve cryptography.
//
// State is currently in-memory, so a restart loses the anti-replay set and the
// collected-batch record. Both losses are privacy failures rather than
// availability ones; see docs/THREAT_MODEL.md.
//
// » THE TWO CONSEQUENCES OF THE IN-MEMORY STORE, spelled out, because they are
// » not rough edges:
// »   1. A restart empties the anti-replay set, so replays become possible across
// »      restarts.
// »   2. A restart loses the record of which batches were already collected, so
// »      the no-overlap rule stops being enforced — which re-enables the
// »      differencing attack.
// » Neither is a rough edge; both are privacy failures. See« EXERCISE 45».
type Store struct {
	mu sync.Mutex

	// seen is the anti-replay set: report IDs already accepted.
	seen map[ReportID]struct{}

	// aggregate is the running aggregate share, updated as reports are verified.
	aggregate prio3.AggShare

	// reportIDs are the IDs contributing to the current batch, for the checksum.
	reportIDs []ReportID

	// collected records batch intervals already served, for the overlap check.
	collected []Interval
}

// NewStore returns an empty store with a zero aggregate of the given length.
func NewStore(outputLen int) *Store {
	return &Store{
		seen:      make(map[ReportID]struct{}),
		aggregate: make(prio3.AggShare, outputLen),
	}
}

// Accept records a report ID, rejecting replays.
//
// TODO(week3): implement.
func (s *Store) Accept(id ReportID) error {
	// EXERCISE-BEGIN
	// ─── EXERCISE 42: anti-replay, and why it is a privacy control ───────────
	// Task: under the lock, reject an ID already in `seen`, otherwise insert it.
	//
	// WHY IT MATTERS, and it is not the obvious reason. The naive framing is
	// "stop double-counting", which sounds like a data-quality issue. The real
	// reason is a privacy attack:
	//
	//   A device's true value is 1. Submit its report 1000 times. The aggregate
	//   moves by 1000 instead of 1. Now the attacker has AMPLIFIED one device's
	//   contribution far above the DP noise floor, and can read that device's
	//   value straight out of a published aggregate that is nominally ε-DP.
	//
	// So replay protection is not a correctness nicety sitting next to the
	// privacy machinery — it is one of the mechanisms the ε guarantee depends on.
	// Note how it composes with the contribution bound in EXERCISE 32: both exist
	// to make "one device moves the aggregate by at most Δ" true, and DP is
	// meaningless without it.
	//
	// Task B: the set grows without bound. Reports outside the task's collection
	//         window can never be collected, so their IDs can be forgotten — but
	//         work out how long "can never" actually is (it involves the maximum
	//         batch interval, the clock-skew allowance from EXERCISE 39, and the
	//         collection deadline) and be conservative, because forgetting too
	//         early re-opens the attack above.
	//         Then implement time-bucketed eviction and expose the set size as a
	//         gauge. An unbounded map on a public ingest path is how aggregators
	//         OOM, and you already have the bounded-queue instinct for this from
	//         internal/pipeline.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return ErrTODO
}

// Aggregate adds a verified output share into the running aggregate.
//
// TODO(week3): implement — Accept must have succeeded for this report first, and
// PrepNext must have accepted it. Aggregating an unverified share is the exact
// failure Prio exists to prevent«; see EXERCISE 25».
func (s *Store) Aggregate(id ReportID, out prio3.OutputShare) error {
	return ErrTODO
}

// Snapshot returns the aggregate share and report set for a batch selector,
// enforcing the batch rules.
//
// TODO(week3): implement.
func (s *Store) Snapshot(sel BatchSelector, minBatchSize int) (prio3.AggShare, []ReportID, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 43: the batch rules — the other half of the privacy story ──
	// Prio3 hides individual reports; these three rules stop the AGGREGATES from
	// giving them away again. Implement all three.
	//
	// RULE 1 — MINIMUM BATCH SIZE. Refuse a collection covering fewer than
	//   minBatchSize reports, with ErrBatchTooSmall. Otherwise a collector
	//   requests a one-second interval containing exactly one device's report and
	//   gets that device's value with only DP noise in the way.
	//   Then work out the number: what minimum batch size makes a Prio3Histogram
	//   at ε = 2 give usable data? Use dp.Histogram.PerBucketStdDev
	//   (EXERCISE 5) — the answer connects the batch rule to the DP parameter,
	//   and neither one makes sense chosen alone. That connection is the insight.
	//
	// RULE 2 — NO OVERLAP. Refuse a batch intersecting one already collected,
	//   with ErrBatchOverlap. This is the differencing attack from EXERCISE 26:
	//   collect [0,10) and [0,11) and subtract. Note that the minimum batch size
	//   does NOT protect against this — both batches are large — so you need both
	//   rules, and each one alone gives a false sense of security.
	//
	// RULE 3 — COLLECT ONCE. Refuse a second collection of the same batch. Each
	//   collection spends ε (dp.Ledger) and adds independent noise, so repeated
	//   collections of the same batch let the collector average the noise away.
	//   Ten collections cut the effective noise by √10 and the ledger is the only
	//   thing that noticed.
	//
	// Task: implement, and write a test for each rule that would FAIL if the rule
	//       were removed. Rule tests that pass whether or not the rule exists are
	//       the most common way a control silently stops working.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, nil, ErrTODO
}

// MarkCollected records that a batch has been served, for RULE 2 and RULE 3.
//
// TODO(week3): implement.
func (s *Store) MarkCollected(sel BatchSelector) error { return ErrTODO }

// SeenCount reports the size of the anti-replay set, for the gauge.
func (s *Store) SeenCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.seen)
}

// EvictBefore forgets report IDs older than t.
//
// TODO(week3): implement«, see EXERCISE 42 Task B». Be conservative about
// what "old enough" means.
func (s *Store) EvictBefore(t time.Time) int { return 0 }

// EXERCISE-BEGIN
// ─── EXERCISE 44: make the store an interface ───────────────────────────────
// Everything above is in-memory. Before you make it durable, make it swappable:
//
//   type ReportStore interface {
//       Accept(ReportID) error
//       Aggregate(ReportID, prio3.OutputShare) error
//       Snapshot(BatchSelector, int) (prio3.AggShare, []ReportID, error)
//       MarkCollected(BatchSelector) error
//   }
//
// Task: extract the interface, keep this type as the in-memory implementation,
//       and make leader.go and helper.go depend on the interface. Same move
//       internal/pipeline already makes with its Sink interface — and the payoff
//       is the same: the aggregation logic becomes testable without a database,
//       which is what makes the end-to-end test in e2e_test.go possible at all.
// ─────────────────────────────────────────────────────────────────────────────

// ─── EXERCISE 45 (week 4): durability, and what it is really for ─────────────
// The in-memory store loses the anti-replay set and the collected-batch record on
// restart, and both losses are privacy failures rather than availability ones.
//
// Task A: persist the collected-batch record and the ε ledger. Order matters and
//         the reasoning is write-ahead logging: record the spend and the
//         collection BEFORE returning the aggregate to the collector. If you
//         crash after recording and before responding, the collector retries and
//         gets an error — annoying. If you crash after responding and before
//         recording, the collector can collect the same batch again — a privacy
//         violation. Always fail in the direction that costs availability rather
//         than privacy, and say so explicitly in docs/PRIVACY.md; it is a good
//         articulation of a principle interviewers listen for.
// Task B: persist the anti-replay set. Note it has different requirements: huge,
//         write-heavy, and only needs bounded-time membership. A Bloom filter is
//         tempting — work out whether false positives are acceptable. (A false
//         positive rejects an honest report. Is losing 0.1% of reports
//         acceptable? It depends on whether the loss is INDEPENDENT of the
//         report's content — and with a Bloom filter keyed on a random ID it is,
//         which makes it merely a small uniform sampling loss rather than a bias.
//         That is a genuinely interesting answer; make sure you can give it.)
// Task C: the aggregate share itself. If the leader crashes mid-batch, is the
//         partial aggregate recoverable, or does the batch restart? Note you
//         cannot re-derive it from the reports unless you stored them — which you
//         deliberately did not, for privacy reasons (EXERCISE 18). So the choice
//         is checkpointing the aggregate, or losing the batch. Pick one and
//         justify it.
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END
