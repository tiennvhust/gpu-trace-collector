package dap

import (
	"context"
	"net/http"
	"time"

	"github.com/tiennvhust/gpu-trace-collector/internal/dp"
)

// » THE LEADER IS THE INTERESTING SERVICE TO BUILD, and it is the one that plays
// » to what you already know. It is a stateful coordinator with three concurrent
// » concerns:
// »
// »   INGEST      accept uploads at fleet rate, deduplicate, buffer.
// »   AGGREGATION drive batched round trips to a helper run by someone else, over
// »               the public internet, which will be slow and will fail.
// »   COLLECTION  serve aggregates, enforce the batch rules, spend ε.
// »
// » Every problem the plaintext path already solved shows up again here in a
// » harder form: bounded queues, backpressure, retry with jitter, graceful
// » shutdown, idempotency. Reuse the instincts — internal/pipeline's bounded
// » queue is the right shape for the aggregation job driver, and for the same
// » reason (an unbounded buffer of pending reports converts helper downtime into
// » an OOM).
// »
// » The genuinely new problem: the helper is a DIFFERENT ORGANISATION. You cannot
// » deploy a fix to it, you cannot read its logs, it will have different capacity
// » than you, and its operator's on-call rotation is not yours. Design for that
// » explicitly — every cross-org call needs a timeout, a retry budget, a circuit
// » breaker and a metric, and every error it returns needs to be
// » machine-readable (which is why DAP mandates RFC 9457 problem details).

// Leader is the DAP leader aggregator.
type Leader struct {
	tasks   TaskSet
	stores  map[TaskID]*Store
	keys    map[HPKEConfigID]*HPKEKeypair
	ledger  *dp.Ledger
	client  *http.Client
	metrics *Metrics
}

// LeaderConfig configures a leader.
type LeaderConfig struct {
	Tasks TaskSet
	Keys  map[HPKEConfigID]*HPKEKeypair

	// HelperTimeout bounds each cross-organisation request.
	//
	// » Must be set. A missing timeout on a cross-org call means one slow helper
	// » can pin every goroutine in the aggregation driver until the process is
	// » wedged, and there is nobody you can page to fix it.
	HelperTimeout time.Duration

	// MaxReportsPerJob is the aggregation batch size.
	MaxReportsPerJob int

	// PendingQueueCapacity bounds the number of reports buffered awaiting
	// aggregation. When full, uploads are rejected with 503 and the client
	// retries — the same reject-over-buffer choice internal/pipeline makes.
	PendingQueueCapacity int
}

// NewLeader constructs a leader.
//
// TODO(week3): implement.
func NewLeader(cfg LeaderConfig, m *Metrics) (*Leader, error) { return nil, ErrTODO }

// Routes returns the leader's HTTP handlers.
//
// » DAP's paths are versioned in the URL (/dap/v<n>/...) — check the current
// » draft for the exact shape, and note that clients in the field will be running
// » an older version than your server for as long as the deployment exists, which
// » makes the version prefix load-bearing rather than cosmetic.
func (l *Leader) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks/{task_id}/reports", l.handleUpload)
	mux.HandleFunc("PUT /tasks/{task_id}/collection_jobs/{job_id}", l.handleCollectionJobCreate)
	mux.HandleFunc("POST /tasks/{task_id}/collection_jobs/{job_id}", l.handleCollectionJobPoll)
	mux.HandleFunc("GET /hpke_config", l.handleHPKEConfig)
	return mux
}

// handleUpload accepts one report from a client.
//
// TODO(week3): implement.
func (l *Leader) handleUpload(w http.ResponseWriter, r *http.Request) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 48: the upload path ───────────────────────────────────────
	// The hot path. Everything you learned building the OTLP receiver applies.
	//
	// Order the checks CHEAPEST-FIRST, because rejection cost is what an attacker
	// controls:
	//   1. Content-Type == MediaTypeReport, and a body-size limit
	//      (http.MaxBytesReader) — reject before reading, never after.
	//   2. Task lookup → ProblemUnrecognizedTask.
	//   3. Decode the report → ProblemInvalidMessage.
	//   4. Timestamp within [now − ReportExpiry, now + skew] →
	//      ProblemReportTooEarly / ProblemReportRejected.
	//   5. Anti-replay check → ProblemReportRejected. Note this is deliberately
	//      AFTER decoding, because you need the ID to check it, which means
	//      decoding is attacker-reachable and therefore must be fuzzed
	//      (EXERCISE 38).
	//   6. Enqueue for aggregation → 503 with Retry-After when the queue is full.
	//
	// Return 201 as soon as the report is durably enqueued, NOT after aggregation.
	// The client is on a battery and must not wait for a cross-org round trip.
	// This is the same accept-then-process split as the plaintext path, and the
	// same consequence: once you have said 201, losing the report is invisible to
	// the client, so the queue must be bounded and its drops must be counted.
	//
	// Metrics, with a reason label on every rejection — the money label, exactly
	// as in internal/obs.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	writeProblem(w, &Problem{Type: ProblemInvalidMessage, Status: 501, Detail: "EXERCISE 48"})
}

// handleCollectionJobCreate starts a collection job.
//
// TODO(week3): implement.
func (l *Leader) handleCollectionJobCreate(w http.ResponseWriter, r *http.Request) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 49: collection as an asynchronous job ─────────────────────
	// Collection cannot be synchronous: the leader has to ask the helper for its
	// aggregate share, which is a cross-org round trip that may take minutes and
	// may fail. DAP models it as PUT-to-create, POST-to-poll — the standard
	// long-running-operation pattern.
	//
	// Task: implement PUT. Validate the batch selector against the task's query
	//       type, run the batch rules (EXERCISE 43), and return 201 with the job
	//       in a pending state.
	//
	// IDEMPOTENCY. The job ID is CLIENT-CHOSEN, so a retried PUT with the same ID
	//       and the same query must not create a second job — and a retried PUT
	//       with the same ID and a DIFFERENT query must be rejected with 409, not
	//       silently overwrite. Get the second case right: silently accepting it
	//       would let a collector mutate a job's batch after the ε was spent,
	//       which is a way to get two different aggregates for one ε charge.
	//       You already implemented idempotency reasoning on the produce path;
	//       this is the same discipline with a privacy consequence attached.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	writeProblem(w, &Problem{Type: ProblemInvalidMessage, Status: 501, Detail: "EXERCISE 49"})
}

// handleCollectionJobPoll returns a collection once it is ready.
//
// TODO(week3): implement — 200 with the Collection when done, 202 while pending.
func (l *Leader) handleCollectionJobPoll(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, &Problem{Type: ProblemInvalidMessage, Status: 501, Detail: "EXERCISE 49"})
}

// handleHPKEConfig publishes the leader's current HPKE configs.
//
// TODO(week3): implement — return every ACTIVE config, not just the newest, so
// clients mid-rotation can still upload. Set a Cache-Control max-age that is
// shorter than your key retention window; if clients cache longer than you retain,
// their reports become undecryptable and silently vanish.
func (l *Leader) handleHPKEConfig(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, &Problem{Type: ProblemInvalidMessage, Status: 501, Detail: "EXERCISE 41"})
}

// RunAggregationJobs drives batched aggregation with the helper until ctx is
// cancelled. Started as a goroutine by cmd/dap-leader.
//
// TODO(week3): implement.
func (l *Leader) RunAggregationJobs(ctx context.Context) error {
	// EXERCISE-BEGIN
	// ─── EXERCISE 50: the aggregation driver ────────────────────────────────
	// The loop: take up to MaxReportsPerJob pending reports, PrepInit each one
	// locally, send an AggregationJobInitReq to the helper, combine the prep
	// shares, PrepNext, aggregate the survivors.
	//
	// The engineering, which is the real content of this exercise:
	//
	//   TIMEOUT + RETRY. Bound every helper call and retry with exponential
	//     backoff AND JITTER. Without jitter, a helper restart produces a
	//     thundering herd from every leader replica at once — the failure mode
	//     the AWS Builders' Library article on the plaintext path describes.
	//   IDEMPOTENT JOBS. A retry must not double-count. Aggregation job IDs are
	//     the mechanism: the helper recognises a repeated job ID and returns its
	//     original answer rather than reprocessing. Note this puts the helper's
	//     response in ITS store, so the helper needs job-level memory too.
	//   PARTIAL FAILURE. Reports the helper rejects are dropped, not retried, and
	//     they must be dropped by BOTH sides or the checksum (EXERCISE 40) will
	//     fail at collection time. Getting this wrong means a batch that
	//     aggregates fine and then cannot be collected, which is a maddening bug
	//     to diagnose after the fact — write the test now.
	//   BOUNDED CONCURRENCY. A fixed number of in-flight jobs, sized against the
	//     PrepState footprint (see prio3.PrepState — 5,000 concurrent states at
	//     8 KiB is 40 MiB, and that estimate is the point of the Size method).
	//   CIRCUIT BREAKER. When the helper has been failing for a while, stop
	//     trying and shed uploads early rather than filling the pending queue with
	//     work that cannot complete.
	//
	// This function is the strongest single piece of evidence in Project A that
	// you can build privacy infrastructure rather than just implement a paper.
	// Give it the attention it deserves, and write the failure-injection tests: a
	// helper that times out, one that returns 500, one that rejects half the
	// batch, one that returns a checksum mismatch.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return ErrTODO
}

// writeProblem writes an RFC 9457 problem-details response.
//
// TODO(week3): implement — Content-Type: application/problem+json, the status
// from p.Status, and the JSON body. Log at the right level: a client error is not
// a leader error, and paging on ProblemInvalidMessage would page you on every
// buggy client in the fleet.
func writeProblem(w http.ResponseWriter, p *Problem) {
	w.WriteHeader(http.StatusNotImplemented)
}
