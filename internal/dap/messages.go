package dap

import "time"

// » Every type here has an Encode and a Decode, and both must agree with the
// » draft byte for byte, because the leader and the helper are DIFFERENT
// » PROCESSES RUN BY DIFFERENT ORGANISATIONS. There is no shared library to keep
// » them in sync — the specification is the contract. That is unusual and worth
// » noticing: most of your career, wire compatibility is guaranteed by both sides
// » importing the same generated code. Here it is guaranteed by both sides
// » reading the same English document correctly, which is exactly why the test
// » vectors and the fuzzer matter so much.
// »
// » Work through the draft's §4 message definitions with this file open beside it.

// ReportMetadata is the cleartext envelope on every report.
//
// » NOTE WHAT IS IN CLEAR. The report ID and the timestamp are NOT encrypted:
// » the leader needs the ID to deduplicate and the timestamp to assign the report
// » to a batch, both before it can decrypt anything. So the leader learns WHEN
// » each device reported, and (from the TLS connection) roughly from where.
// »
// » That is real metadata leakage and it belongs in THREAT_MODEL.md as its own
// » row, not folded into "the leader learns nothing". The mitigations are
// » time-precision rounding (below) and, for the network layer, an oblivious
// » relay — Apple's iCloud Private Relay and OHTTP exist for exactly this. Saying
// » "DAP hides the values but not the fact or timing of reporting, and here is
// » what you add to fix that" is a much stronger answer than claiming the
// » protocol solves everything.
type ReportMetadata struct {
	ID   ReportID
	Time time.Time
}

// TimePrecision rounds report timestamps to a multiple of the task's precision.
//
// » A per-second timestamp on a report is a near-unique identifier when combined
// » with anything else, so DAP rounds to the task's time_precision (say 3600s)
// » and that rounded value defines the batch interval. The trade-off is direct:
// » coarser precision means larger batches, which means better privacy and worse
// » timeliness. One number, two stakeholders, and the argument for the value you
// » pick belongs in docs/PRIVACY.md.
//
// TODO(week3): implement — round DOWN to a multiple of precision, and be careful
// with negative Unix times and with precision == 0.
func TimePrecision(t time.Time, precision time.Duration) time.Time {
	return time.Time{}
}

// HPKECiphertext is one input share, encrypted to one aggregator.
type HPKECiphertext struct {
	ConfigID uint8
	Enc      []byte // the encapsulated key
	Payload  []byte // AEAD ciphertext
}

// Report is what a client uploads to the leader.
//
// » One report, two ciphertexts: [0] for the leader, [1] for the helper. The
// » leader cannot open [1] — that is the entire point, and it is why the client
// » can safely send both to one endpoint. If the leader could open both, the two
// » aggregators would be one aggregator with extra steps.
type Report struct {
	Metadata        ReportMetadata
	PublicShare     []byte
	EncryptedShares [2]HPKECiphertext
}

// Encode serialises the report per draft §4.5.
//
// TODO(week3): implement.
func (r *Report) Encode() ([]byte, error) { return nil, ErrTODO }

// DecodeReport parses a report.
//
// TODO(week3): implement.
func DecodeReport(b []byte) (*Report, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 39: decode defensively ────────────────────────────────────
	// Task: decode in the draft's field order, and reject rather than repair:
	//   - exactly 2 encrypted shares, no more and no fewer;
	//   - no trailing bytes (Decoder.Done());
	//   - a timestamp that is absurdly far in the future — DAP has a specific
	//     error for this, ProblemReportTooEarly, and the reason is subtle and
	//     worth understanding: a report timestamped a year ahead would land in a
	//     batch that cannot be collected until then, so accepting it lets a
	//     client pin storage in the aggregator indefinitely. A resource-exhaustion
	//     attack that arrives dressed as a clock-skew bug.
	//
	// Then decide the clock-skew policy, because you have to: how far ahead is
	// "absurd"? Devices have bad clocks. Rejecting anything ahead of now drops
	// legitimate reports from phones whose NTP has not run; accepting a week
	// ahead means a week of pinned storage. Pick a number, put it in config, and
	// write the reasoning in docs/PRIVACY.md. Silent data loss from clock skew is
	// a genuinely common production failure in device telemetry and you have
	// probably seen it.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// ─── aggregation: leader ⇄ helper ────────────────────────────────────────────

// PrepareInit is one report's worth of work sent from leader to helper.
type PrepareInit struct {
	Metadata    ReportMetadata
	PublicShare []byte
	// Payload is the helper's encrypted input share, relayed by the leader.
	Payload HPKECiphertext
	// LeaderPrep is the leader's prep share for this report.
	LeaderPrep []byte
}

// AggregationJobInitReq starts an aggregation job: a batch of reports the two
// aggregators verify together.
//
// » BATCHED, not per-report, and the reason is arithmetic. One HTTP round trip
// » per report at 10^5 reports/second across two organisations' networks is
// » 10^5 requests/second of cross-org traffic — with TLS handshakes, connection
// » pools and per-request overhead dominating the actual cryptography. Batching
// » 1000 reports per job cuts that by three orders of magnitude.
// »
// » The cost is latency and failure granularity: a job's reports all wait for the
// » slowest, and a partial failure needs per-report status in the response (which
// » is why AggregationJobResp carries a status per report rather than one status
// » for the job). This is the same batching trade-off as the OTLP exporter's
// » batch processor on the plaintext path, and the same reasoning applies.
type AggregationJobInitReq struct {
	AggregationParameter []byte
	PartialBatchSelector BatchSelector
	PrepareInits         []PrepareInit
}

// Encode serialises the request.
//
// TODO(week3): implement.
func (a *AggregationJobInitReq) Encode() ([]byte, error) { return nil, ErrTODO }

// DecodeAggregationJobInitReq parses the request.
//
// TODO(week3): implement — bound len(PrepareInits) against a configured maximum
// before allocating the slice. See the note on Decoder.Vec32.
func DecodeAggregationJobInitReq(b []byte) (*AggregationJobInitReq, error) {
	return nil, ErrTODO
}

// PrepareRespState is the per-report outcome in an aggregation job response.
type PrepareRespState uint8

// Prepare response states.
const (
	PrepareStateContinue PrepareRespState = 0
	PrepareStateFinished PrepareRespState = 1
	PrepareStateReject   PrepareRespState = 2
)

// PrepareResp is one report's outcome.
//
// » Per-report, so a single malformed report does not fail a job of a thousand.
// » Getting this granularity right is what makes the aggregation pipeline robust
// » against a buggy client rollout, which on a fleet of a billion devices is not
// » a hypothetical.
type PrepareResp struct {
	ReportID ReportID
	State    PrepareRespState
	Payload  []byte // prep share, when State is Continue
	Error    string // rejection reason, for metrics; not in the wire format
}

// AggregationJobResp is the helper's answer.
type AggregationJobResp struct {
	PrepareResps []PrepareResp
}

// Encode serialises the response.
//
// TODO(week3): implement.
func (a *AggregationJobResp) Encode() ([]byte, error) { return nil, ErrTODO }

// DecodeAggregationJobResp parses the response.
//
// TODO(week3): implement.
func DecodeAggregationJobResp(b []byte) (*AggregationJobResp, error) { return nil, ErrTODO }

// ─── collection: collector ⇄ aggregators ─────────────────────────────────────

// Interval is a half-open time interval [Start, Start+Duration).
type Interval struct {
	Start    time.Time
	Duration time.Duration
}

// Contains reports whether t falls in the interval.
//
// TODO(week3): implement — half-open, so Start is in and Start+Duration is out.
// Off-by-one here means a report counted in two batches, which is a privacy bug
// (see the differencing attack in internal/vdaf/prio3/driver.go,« EXERCISE 26»),
// not merely a counting bug.
func (i Interval) Contains(t time.Time) bool { return false }

// BatchSelector identifies which reports a collection covers.
//
// » Two query types in DAP, and the difference is a privacy decision:
// »   TIME INTERVAL — "all reports timestamped in [t, t+d)". Predictable and easy
// »     to reason about, but the collector chooses the boundaries, and a collector
// »     that can choose overlapping intervals can mount the differencing attack.
// »     Hence the no-overlap rule in batch.go.
// »   FIXED SIZE — "the next N reports the leader has, whichever they are". The
// »     leader assigns reports to batches, so the collector cannot target a
// »     specific device's window. Better privacy, less predictable data.
// »
// » Which one you would choose for gpu-trace, and why, is a good design-doc
// » paragraph (§7 of the plan).
type BatchSelector struct {
	QueryType QueryType
	// Interval is set for QueryTypeTimeInterval.
	Interval Interval
	// BatchID is set for QueryTypeFixedSize.
	BatchID [IDLen]byte
}

// QueryType selects the batch addressing mode.
type QueryType uint8

// Query types.
const (
	QueryTypeTimeInterval QueryType = 1
	QueryTypeFixedSize    QueryType = 2
)

// CollectionReq is the collector's request to the leader.
type CollectionReq struct {
	Query                BatchSelector
	AggregationParameter []byte
}

// Collection is the leader's response, once the batch is complete.
type Collection struct {
	PartialBatchSelector BatchSelector
	ReportCount          uint64
	Interval             Interval
	// LeaderEncryptedAggShare and HelperEncryptedAggShare are each encrypted to
	// the COLLECTOR's HPKE key, so the leader cannot read the helper's share
	// even though it relays it.
	LeaderEncryptedAggShare HPKECiphertext
	HelperEncryptedAggShare HPKECiphertext
}

// » READ THAT LAST FIELD AGAIN, because it is the design's cleverest small move.
// » The helper's aggregate share travels THROUGH the leader, encrypted to the
// » collector. So:
// »   - the collector makes one request to one endpoint;
// »   - the helper still needs no public ingress;
// »   - and the leader relays a share it cannot read, so it still cannot
// »     reconstruct the aggregate.
// » One party carries another party's secret without being trusted with it. That
// » pattern — route through the convenient party, encrypt to the entitled one —
// » is worth stealing for other designs.

// Encode serialises the collection.
//
// TODO(week3): implement.
func (c *Collection) Encode() ([]byte, error) { return nil, ErrTODO }

// DecodeCollection parses a collection.
//
// TODO(week3): implement.
func DecodeCollection(b []byte) (*Collection, error) { return nil, ErrTODO }

// AggregateShareReq is the leader asking the helper for its aggregate share.
type AggregateShareReq struct {
	BatchSelector        BatchSelector
	AggregationParameter []byte
	ReportCount          uint64
	Checksum             [32]byte
}

// » THE CHECKSUM IS THE MOST INTERESTING FIELD IN THE PROTOCOL. Take a minute on
// » it.
// »
// » Both aggregators must aggregate EXACTLY THE SAME SET of reports. If the
// » leader includes a report the helper does not, the two aggregate shares are
// » shares of different sums, and adding them gives noise — not an error, just a
// » silently wrong answer. Worse, the difference between the two sets is exactly
// » the information a malicious leader would need: by including one extra report
// » in its own share and telling the helper a different set, a leader could
// » isolate a single device's contribution.
// »
// » So the request carries a commitment to the set: the report count plus a
// » checksum over the report IDs. The helper recomputes both from its own view and
// » refuses if they differ. Two independent parties agreeing on set membership
// » without exchanging the set — with the sharp constraint that the commitment
// » must be order-independent, since the two aggregators may have processed the
// » reports in any order. (The draft uses XOR of per-ID hashes for exactly that
// » reason. Note that XOR is order-independent AND self-cancelling, so think about
// » whether a duplicate ID is possible here and what it would do.)
// »
// » This is a distributed-systems problem wearing a cryptography hat, and it is
// » the single best thing in this project to bring up when someone asks what you
// » found interesting about DAP.

// Checksum computes the order-independent commitment over a set of report IDs.
//
// TODO(week3): implement.
func Checksum(ids []ReportID) [32]byte {
	// EXERCISE-BEGIN
	// ─── EXERCISE 40: the batch checksum ────────────────────────────────────
	// Task: per the draft, XOR the SHA-256 of each report ID.
	//
	// Then answer these, in docs/PRIVACY.md, because they are the whole reason
	// the field exists:
	//   1. Why not sort the IDs and hash the concatenation? (Consider that the
	//      leader streams reports as they arrive and must be able to update the
	//      commitment incrementally, in O(1) memory, without holding the set.)
	//   2. XOR is self-cancelling: the same ID included twice contributes
	//      nothing. Is that exploitable? Trace through what the anti-replay set
	//      in store.go guarantees before answering — and note that the answer
	//      depends on a property enforced in a different package, which is
	//      exactly the kind of cross-component reasoning that makes this hard to
	//      get right.
	//   3. What does the helper do when the checksums differ? There is no way to
	//      tell WHICH reports disagree. The batch is unrecoverable, so the honest
	//      answer is "fail the batch and alert" — but then work out what a
	//      malicious leader gains by deliberately causing mismatches, and what it
	//      costs the task's availability.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return [32]byte{}
}
