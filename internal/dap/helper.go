package dap

import "net/http"

// » THE HELPER IS SIMPLE, AND THAT SIMPLICITY IS A DESIGN GOAL RATHER THAN AN
// » ACCIDENT. It is stateless between jobs apart from its aggregate share and its
// » anti-replay set; it has no public ingress (only the leader calls it); it does
// » no coordination.
// »
// » Why deliberately: someone has to be persuaded to run it. The whole privacy
// » model rests on the two aggregators being different organisations that do not
// » collude, so the helper role has to be cheap enough that a non-profit, a
// » standards body or a privacy-focused partner will operate it. If being a helper
// » required a large engineering team, there would be no helpers, and the protocol
// » would have no security. ISRG's Divvi Up exists precisely to be this party.
// »
// » Notice that this is a SOCIAL constraint expressed as an architectural one. The
// » seed trick in prio3« EXERCISE 19» is the same story from another angle: the
// » helper receives 16 bytes per report where the leader receives kilobytes, so
// » the cheap role is also the low-bandwidth role. Being able to articulate that —
// » that the protocol's shape is driven by who can be convinced to run each part —
// » is a much more interesting thing to say about DAP than reciting its endpoints.

// Helper is the DAP helper aggregator.
type Helper struct {
	tasks   TaskSet
	stores  map[TaskID]*Store
	keys    map[HPKEConfigID]*HPKEKeypair
	jobs    map[[IDLen]byte]*AggregationJobResp
	metrics *Metrics
}

// NewHelper constructs a helper.
//
// TODO(week3): implement.
func NewHelper(tasks TaskSet, keys map[HPKEConfigID]*HPKEKeypair, m *Metrics) (*Helper, error) {
	return nil, ErrTODO
}

// Routes returns the helper's HTTP handlers.
//
// » Only the leader calls these. Authenticate it — mutual TLS, or a bearer token
// » — and say which you chose in THREAT_MODEL.md. An unauthenticated helper is a
// » free aggregation oracle: anyone can submit an aggregation job for a task and
// » learn whether reports validate, which leaks more than it looks like.
func (h *Helper) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /tasks/{task_id}/aggregation_jobs/{job_id}", h.handleAggregationJob)
	mux.HandleFunc("POST /tasks/{task_id}/aggregate_shares", h.handleAggregateShare)
	mux.HandleFunc("GET /hpke_config", h.handleHPKEConfig)
	return mux
}

// handleAggregationJob processes one batch of reports for the leader.
//
// TODO(week3): implement.
func (h *Helper) handleAggregationJob(w http.ResponseWriter, r *http.Request) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 51: the helper's aggregation step ──────────────────────────
	// For each PrepareInit in the request:
	//   1. anti-replay check on the report ID (the helper keeps its OWN set — do
	//      not trust the leader to have done it, because the leader is exactly the
	//      party whose replays you are defending against);
	//   2. HPKE-Open the input share with the correct AAD;
	//   3. PrepInit → the helper's prep share;
	//   4. combine with the leader's prep share from the request;
	//   5. PrepNext → accept or reject;
	//   6. aggregate the accepted output share.
	//
	// Return a per-report status. One bad report must not fail the job.
	//
	// IDEMPOTENCY, and it is not optional: the leader retries. Cache the response
	// by aggregation job ID and return the cached answer for a repeated ID. Then
	// handle the nastier case — a repeated job ID with DIFFERENT report contents.
	// That is either a leader bug or a leader attack; reject with a distinct
	// problem type and count it separately, because those two possibilities have
	// very different responses and you will want the evidence.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	writeProblem(w, &Problem{Type: ProblemInvalidMessage, Status: 501, Detail: "EXERCISE 51"})
}

// handleAggregateShare returns the helper's aggregate share for a batch.
//
// TODO(week3): implement.
func (h *Helper) handleAggregateShare(w http.ResponseWriter, r *http.Request) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 52: where the helper says no ──────────────────────────────
	// This is the helper's one moment of real power, and the reason the trust
	// model works at all. Before releasing its aggregate share it must
	// INDEPENDENTLY verify:
	//   1. the report count meets the task's MinBatchSize (do not trust the
	//      leader's count — recompute from its own store);
	//   2. the checksum matches its own view of the batch (EXERCISE 40);
	//   3. this batch has not been collected before, per its own record;
	//   4. the batch does not overlap one it has already served.
	//
	// A malicious leader trying the differencing attack from prio3 EXERCISE 26 is
	// stopped HERE, by a second party checking the same rules from its own state.
	// That is the whole architecture in one handler: the privacy property comes
	// from two parties independently enforcing the same rules on independently
	// maintained state, not from cryptography.
	//
	// Write the test that matters: construct a leader that requests an overlapping
	// batch and assert the helper refuses. Then construct one that lies about the
	// report count and assert the same. If the helper's checks were merely a copy
	// of the leader's checks on the leader's data, they would be worthless — make
	// sure your test would catch that.
	//
	// Then encrypt the share to the COLLECTOR's public key, not the leader's. The
	// leader relays it and must not be able to read it; see the note on
	// Collection in messages.go.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	writeProblem(w, &Problem{Type: ProblemInvalidMessage, Status: 501, Detail: "EXERCISE 52"})
}

// handleHPKEConfig publishes the helper's HPKE configs.
//
// TODO(week3): implement — see the leader's equivalent.
func (h *Helper) handleHPKEConfig(w http.ResponseWriter, r *http.Request) {
	writeProblem(w, &Problem{Type: ProblemInvalidMessage, Status: 501, Detail: "EXERCISE 41"})
}
