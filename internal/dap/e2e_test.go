package dap

import "testing"

// » THE TEST THAT PROVES PROJECT A WORKS. Everything else in this repository
// » tests a layer; this drives the whole protocol in-process: client → leader →
// » helper → collector, with real HTTP over httptest servers and no mocks of the
// » protocol itself.
// »
// » Build it in week 3 and keep it fast (under a second). It is the test you will
// » run hundreds of times, and it is the one to demo.

func TestEndToEndPrio3Count(t *testing.T) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 59: the end-to-end test ───────────────────────────────────
	// Structure:
	//   1. Generate HPKE keypairs for the leader, the helper and the collector.
	//   2. Build a Task: Prio3Count, MinBatchSize 10, ε = 1, TimePrecision 1h.
	//   3. Start the helper on an httptest.Server; start the leader on another,
	//      pointed at the helper's URL. REAL HTTP, not a function call — it is the
	//      only way to exercise the codec, the media types and the error mapping,
	//      and those are where the bugs are.
	//   4. Upload 20 reports, 15 true and 5 false.
	//   5. Drive one aggregation job.
	//   6. Collect, and assert the noised result is 15 ± a tolerance derived from
	//      ε. NOT an exact equality — see below.
	//
	// THE TOLERANCE IS THE INTERESTING PART, and getting it right is most of the
	// value of this exercise. At ε = 1 with sensitivity 1, the Laplace noise has
	// standard deviation √2. A 5σ window is about ±7, so assert |result − 15| < 7
	// and derive that 7 in the test from ε rather than hardcoding it. A test that
	// asserts exact equality is testing that the noise is broken.
	//
	// Then a SEPARATE test with the noise mechanism swapped for a no-op that
	// asserts EXACT equality with 15 — that one catches arithmetic bugs the noisy
	// test cannot see. Two tests, two purposes: one proves the aggregation is
	// correct, the other proves the noise is present. Do not try to make one test
	// do both, because the tolerance that accommodates noise also hides
	// off-by-one errors in the aggregation.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	t.Fatalf("EXERCISE 59: implement the end-to-end test")
}

func TestEndToEndPrio3Histogram(t *testing.T) {
	// » Same shape, 8 buckets, 100 reports with a known distribution. Assert every
	// » bucket is within tolerance and that the bucket counts sum to roughly 100.
	t.Fatalf("EXERCISE 59: extend the end-to-end test to Prio3Histogram")
}

func TestReplayedReportIsRejected(t *testing.T) {
	// » Upload the same report ID twice; the second must be rejected and the
	// » aggregate must move by ONE, not two. Assert on the aggregate, not just on
	// » the HTTP status — a leader that returns 400 and aggregates anyway passes a
	// » status-only test, and that is the bug that matters. See« EXERCISE 42» for why
	// » this is a privacy control rather than a counting one.
	t.Fatalf("EXERCISE 42: implement the replay test")
}

func TestBatchBelowMinimumIsRefused(t *testing.T) {
	// » Upload 5 reports against a task with MinBatchSize 10 and assert the
	// » collection fails with ErrBatchTooSmall. Then upload 5 more and assert it
	// » succeeds — a rule that always refuses is not a rule.
	t.Fatalf("EXERCISE 43: implement the minimum-batch-size test")
}

func TestOverlappingBatchIsRefused(t *testing.T) {
	// » THE DIFFERENCING-ATTACK TEST, and the most important negative test in the
	// » repository. Collect [t, t+1h), then attempt [t, t+2h) — which contains
	// » every report from the first batch. It must be refused with ErrBatchOverlap.
	// »
	// » Then write the comment above the assertion explaining the attack, because
	// » in six months the person tempted to relax this rule to fix a "collector
	// » can't re-run a query" complaint needs to find that explanation right here.
	t.Fatalf("EXERCISE 43: implement the batch-overlap test")
}

func TestMaliciousClientReportIsRejected(t *testing.T) {
	// » Robustness, not privacy. Craft a report whose input vector is 10^6 in a
	// » Prio3Count task, aggregate it, and assert both aggregators reject it and
	// » the aggregate is unmoved.
	// »
	// » This is the test that demonstrates the "V" in VDAF actually works, and it is
	// » the one to point at when asked why Prio needs verifiability. Do not skip it
	// » — a happy-path-only test suite cannot distinguish your implementation from
	// » one that never verifies anything at all.
	t.Fatalf("EXERCISE 23: implement the malicious-client test")
}

func TestHelperRefusesOverlappingBatchIndependently(t *testing.T) {
	// » The trust model, tested. Drive the helper directly with an AggregateShareReq
	// » for a batch it has already served, bypassing the leader entirely, and assert
	// » it refuses from its OWN state.
	// »
	// » If this test passes only because the leader checked first, the two-aggregator
	// » model provides nothing — the helper would be a rubber stamp. So construct a
	// » deliberately misbehaving leader here. See« EXERCISE 52».
	t.Fatalf("EXERCISE 52: implement the independent-helper-check test")
}

func BenchmarkUploadPath(b *testing.B) {
	// » The load test from the plan: reports/sec through the leader's upload path,
	// » plus client-side CPU and bytes on wire versus the plaintext OTLP path.
	// »
	// » Run it against BOTH paths on the same machine so the comparison is fair, and
	// » put the table in docs/BENCHMARKS.md. "Quantify the privacy tax" is the
	// » deliverable, and a hiring manager on a billions-of-devices team will look at
	// » that table before reading any of the code.
	b.Skip("EXERCISE 48: implement the upload path, then benchmark it")
}
