package dap

import (
	"context"

	"github.com/tiennvhust/gpu-trace-collector/internal/dp"
	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/prio3"
)

// The collector is the only role that sees a result, so this is where the ε
// budget is spent and where the DP noise is added. There is deliberately exactly
// one code path from aggregate shares to a published number, and the noise is on
// it: unshardExact and applyNoise are unexported so that an exact aggregate
// cannot be obtained from outside this package.
//
// » AND THAT IS THE FILE'S ENTIRE JOB, which makes this the file
// » where the privacy budget is actually spent and where the noise is actually
// » added. Everything upstream is machinery; this is the moment information leaves
// » the system.
// »
// » Keep exactly ONE code path from aggregate shares to a published number, and
// » make the noise unavoidable on it. A second path — a debug endpoint, a test
// » helper promoted to production, an "internal only" metric — is how the noise
// » gets skipped, and it will not be skipped deliberately. It will be skipped by
// » someone six months from now who needed the exact value for a one-off
// » investigation and did not know why the wrapper existed. Design against that
// » person: make Unshard-without-noise impossible to reach from outside the
// » package, not merely discouraged.

// Collector combines aggregate shares from both aggregators, applies DP noise,
// and returns the published aggregate.
type Collector struct {
	task   *Task
	keys   *HPKEKeypair
	ledger *dp.Ledger
	mech   dp.IntMechanism
}

// NewCollector constructs a collector for one task.
//
// TODO(week3): implement — build the IntMechanism from task.DP (discrete
// Gaussian; see the note on internal/dp.IntMechanism for why not the continuous
// one) and register the task's budget with the ledger.
func NewCollector(task *Task, keys *HPKEKeypair, ledger *dp.Ledger) (*Collector, error) {
	return nil, ErrTODO
}

// Result is a published, noised aggregate.
type Result struct {
	// Values are the noised aggregate. Signed, because DP noise is signed and
	// clamping at this layer would bias the statistics;« see prio3 EXERCISE 30».
	Values []int64

	// ReportCount is the number of reports in the batch.
	//
	// » CAREFUL: this is itself a statistic about the population, and publishing
	// » it exactly leaks. If the batch is "all devices that hit a CUDA OOM this
	// » hour", the exact count is exactly the sensitive number. DAP requires the
	// » count for the batch-size check, but whether you PUBLISH it is your
	// » decision — and if you do, it needs its own noise and its own share of the
	// » ε budget. It is very easy to noise the histogram carefully and hand out
	// » the exact n beside it. Decide, and write it in docs/PRIVACY.md.
	ReportCount uint64

	// Interval is the batch's time interval.
	Interval Interval

	// Epsilon is the privacy budget this collection spent.
	//
	// » Return it, do not just log it. The consumer of the data needs to know how
	// » noisy the number is to use it correctly, and a downstream analyst who does
	// » not know ε will over-interpret a noisy bucket. Publishing the privacy
	// » parameters alongside the data is good practice and it is what Apple's own
	// » DP releases do.
	Epsilon float64
}

// Collect runs a collection: request shares from both aggregators, combine,
// noise, publish.
//
// TODO(week3): implement.
func (c *Collector) Collect(ctx context.Context, sel BatchSelector) (*Result, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 53: the collection path, and the order of operations ────────
	// Steps, and the ORDER IS THE EXERCISE:
	//   1. Spend ε from the ledger. FIRST. Before any network call, before
	//      combining anything. If the spend fails, nothing else happens and no
	//      information moves.
	//   2. PUT a collection job to the leader, poll until it completes.
	//   3. HPKE-Open both aggregate shares with the collector's private key.
	//   4. prio3.Unshard → the exact aggregate.
	//   5. Add discrete noise to every element (internal/dp).
	//   6. Return the noised result WITH the ε it cost.
	//
	// Why spend first: if you collect and then discover the budget is exhausted,
	// you are holding the exact aggregate in memory and the only thing preventing
	// disclosure is that you remembered to discard it. Spend-then-collect makes the
	// budget a hard gate instead of a soft check — the same write-ahead reasoning
	// as store.go EXERCISE 45. Fail in the direction that costs availability, not
	// privacy.
	//
	// Task B: what happens if steps 2–4 fail after the spend succeeded? You have
	//   spent ε and got nothing. Refund, or not? Argue it: a refund on failure is
	//   an oracle — retry until the noise happens to be favourable, and you have
	//   defeated the mechanism. So the answer is NO REFUND, and that is
	//   counter-intuitive enough to be a genuinely good interview answer. Make sure
	//   you can also say what it costs operationally (a flaky helper burns budget),
	//   and what you would do about that (retry the same collection job ID, which
	//   is idempotent, rather than starting a new one).
	//
	// Task C: the noise must be sampled with crypto/rand and must be added AFTER
	//   unsharding, at exactly one place. Add a test that asserts two collections
	//   of the same (hypothetical) batch produce different values — proving the
	//   noise is actually being applied and is not a fixed offset. A DP
	//   implementation that adds a constant passes every "the noise is applied"
	//   test that only checks the value changed from the true one.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// applyNoise adds DP noise to an exact aggregate. Unexported deliberately: this
// is the only route from an exact value to a published one.
//
// TODO(week3): implement.
func (c *Collector) applyNoise(exact []uint64) ([]int64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 54: independent noise per element ─────────────────────────
	// Task: draw a FRESH sample for each element. Reusing one sample across a
	//       histogram's buckets is a real and easy mistake — the noise becomes a
	//       constant shift, all differences between buckets are exact, and the
	//       mechanism provides no protection at all while looking noisy.
	//
	// Task B: the sensitivity you pass to the mechanism must be the one the VDAF
	//       circuit actually enforces, not the one you wish it enforced. Wire it
	//       from the FLP's parameters rather than a config value someone can set
	//       independently — the whole failure mode in prio3 EXERCISE 32 is a
	//       config-declared bound diverging from a circuit-enforced one. Making
	//       that divergence impossible to express is better than documenting it.
	//
	// Task C: write the test from prio3 EXERCISE 30 Task C — a true count of 2
	//       with noise of −5. Confirm you publish −3 and not 18446744073709551613.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// unshardExact combines aggregate shares into the exact aggregate. Unexported,
// and it must stay that way — see the note at the top of this file.
//
// TODO(week3): implement.
func (c *Collector) unshardExact(shares []prio3.AggShare, n int) ([]uint64, error) {
	return nil, ErrTODO
}
