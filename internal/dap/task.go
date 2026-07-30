package dap

import (
	"fmt"
	"time"

	"github.com/tiennvhust/gpu-trace-collector/internal/dp"
	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/prio3"
)

// Task is one aggregation task: the complete agreement between the two
// aggregators about what is being measured and under what privacy terms.
//
// » BOTH AGGREGATORS MUST HOLD IDENTICAL TASK PARAMETERS. There is no
// » synchronisation mechanism in base DAP — the two organisations exchange these
// » out of band and configure them separately, which means a typo in one operator's
// » YAML produces a task that fails in a confusing way, or worse, one that
// » succeeds with mismatched privacy parameters.
// »
// » That problem is why draft-ietf-ppm-dap-taskprov exists: it makes task
// » provisioning part of the protocol, deriving the task ID from a hash of the
// » parameters so a mismatch is DETECTED rather than silently tolerated. Read that
// » draft — it is the Apple-co-authored one — and then implement the derived task
// » ID in« EXERCISE 47». It is a small change with a genuinely good story attached:
// » "make the configuration self-verifying instead of trusting two operators to
// » match" is a design instinct worth demonstrating.
type Task struct {
	ID TaskID

	// VDAF is the aggregation function. Both aggregators must instantiate the
	// same variant with the same parameters.
	VDAF prio3.VDAF

	// LeaderURL and HelperURL are the aggregators' base URLs.
	LeaderURL string
	HelperURL string

	// VerifyKey is the aggregators' shared secret for the FLP query randomness.
	// It must be secret from clients —« see prio3 EXERCISE 25».
	VerifyKey []byte

	// TimePrecision is the batch granularity; report timestamps are rounded down
	// to a multiple of it.
	TimePrecision time.Duration

	// MinBatchSize is the smallest number of reports a collection may cover.
	MinBatchSize int

	// MaxBatchQueryCount is how many times one batch may be collected. Anything
	// above 1 spends ε per query and lets the collector average out the noise;
	// see store.go RULE 3.
	MaxBatchQueryCount int

	// QueryType is time-interval or fixed-size batching.
	QueryType QueryType

	// DP is the noise applied to the aggregate before publication, and Budget is
	// the total ε available to this task per period.
	DP     dp.Params
	Budget float64

	// ReportExpiry is how long after its timestamp a report may still be
	// uploaded. Bounds the anti-replay set and the clock-skew tolerance.
	ReportExpiry time.Duration
}

// Validate checks the task for internally inconsistent parameters.
//
// TODO(week3): implement.
func (t *Task) Validate() error {
	// EXERCISE-BEGIN
	// ─── EXERCISE 46: refuse to deploy an unusable task ─────────────────────
	// A task can be individually-valid in every field and collectively useless.
	// Catch that HERE, at startup, rather than discovering it from a dashboard of
	// pure noise three weeks later.
	//
	// Check, at minimum:
	//   - ε > 0, δ ∈ [0, 1), Budget ≥ ε (you cannot afford even one collection
	//     otherwise);
	//   - MinBatchSize > 0, and warn loudly below ~1000: a small batch means a
	//     single device is a large fraction of the aggregate;
	//   - MaxBatchQueryCount == 1 unless the operator has explicitly opted in,
	//     and if > 1, Budget ≥ ε × MaxBatchQueryCount;
	//   - TimePrecision > 0 and not absurdly fine (a 1-second precision on an
	//     hourly-reporting fleet gives batches of ~n/3600, which will fail the
	//     minimum batch size in production and not in your test);
	//   - THE INTERESTING ONE: expected utility. Use
	//     dp.Histogram.PerBucketStdDev (EXERCISE 5) with the task's ε and the
	//     VDAF's output length, and refuse the task if the expected per-bucket
	//     noise exceeds some fraction of MinBatchSize / buckets. A task whose
	//     noise is larger than its signal should not start.
	//
	// Return every problem at once (errors.Join), not just the first. An operator
	// fixing a config file wants the whole list, and this is a place where the
	// small courtesy is very visible.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return ErrTODO
}

// DeriveID computes the task ID from the task's parameters, per the taskprov
// draft, so that two operators with mismatched configuration get different IDs
// and fail loudly instead of aggregating inconsistently.
//
// TODO(week3): implement.
func (t *Task) DeriveID() (TaskID, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 47: self-verifying task configuration ─────────────────────
	// Task: hash a canonical encoding of every privacy-relevant parameter — VDAF
	//       identifier and its parameters, both aggregator URLs, time precision,
	//       minimum batch size, max query count, query type — into the task ID.
	//
	// The payoff: if the helper's config says MinBatchSize 100 and the leader's
	// says 10000, the derived IDs differ and the very first aggregation job fails
	// with "unrecognized task" instead of quietly running at the weaker of the
	// two settings. Configuration mismatch becomes a startup error rather than a
	// privacy incident.
	//
	// Decide deliberately which fields go in. The verify key must NOT (it is
	// secret, and the ID is public). Should the ε? Argue it both ways: including
	// it means an operator cannot change the noise without a new task, which is
	// arguably the point; excluding it means ε is not protected by the mechanism
	// at all. There is a defensible answer either way and the reasoning is the
	// deliverable — this is a good short section for docs/PRIVACY.md.
	//
	// Spec: draft-ietf-ppm-dap-taskprov (S. Wang, Apple; C. Patton, Cloudflare).
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return TaskID{}, ErrTODO
}

// TaskSet is a collection of tasks indexed by ID, loaded at startup.
type TaskSet map[TaskID]*Task

// Lookup returns the task for id.
func (ts TaskSet) Lookup(id TaskID) (*Task, error) {
	t, ok := ts[id]
	if !ok {
		return nil, &Problem{
			Type:   ProblemUnrecognizedTask,
			Status: 400,
			Detail: fmt.Sprintf("no such task %s", id),
			TaskID: id.String(),
		}
	}
	return t, nil
}
