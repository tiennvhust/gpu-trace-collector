// Package prio3 implements the Prio3 family of Verifiable Distributed
// Aggregation Functions from draft-irtf-cfrg-vdaf.
//
// The protocol, in four steps:
//
//  1. SHARD (client) — encode the measurement as a vector of field elements,
//     split it additively into one input share per aggregator, and generate a
//     proof that the encoding is well-formed. Each share alone is uniformly
//     random and reveals nothing.
//  2. PREPARE (both aggregators, one round) — each evaluates the verification
//     circuit on its own share and broadcasts a short prep share. Combining them
//     reveals exactly one bit: whether the input was valid.
//  3. AGGREGATE (both, independently) — add the verified output share into a
//     running total. O(1) memory per batch, so reports are never stored.
//  4. UNSHARD (collector) — add one aggregate share per aggregator to recover the
//     true aggregate, then apply DP noise before publishing (internal/dap).
//
// The proof system is what makes the protocol robust against malicious clients: a
// client cannot submit an out-of-range value to skew the aggregate, and the
// aggregators can check that without seeing the value.
//
// » THIS IS THE HEART OF THE PROJECT, so read the four steps above twice before
// » writing any code. "Privacy Preserving Measurement" is not a generic phrase —
// » PPM is an IETF working group, and Prio3 is what its two core documents
// » specify. Apple engineers co-author them. Getting this package to pass the
// » draft's published test vectors is the single most precisely targeted thing
// » you can do in eight weeks.
//
// » THREE THINGS THE SUMMARY ABOVE UNDERSTATES:
// »
// »   Step 1's shares are uniformly random INDIVIDUALLY. Each aggregator's view
// »   of one report is a uniform field element, so "learns nothing" is literal
// »   rather than approximate — see internal/vdaf/field on why the prime matters.
// »
// »   Step 2 leaks exactly one bit and no more, and that bit is "was the input
// »   valid". Write that in THREAT_MODEL.md rather than claiming zero leakage;
// »   the honest version is more convincing and it is the version an interviewer
// »   is listening for.
// »
// »   Step 3's O(1) memory is a PRIVACY property as much as a scaling one. The
// »   aggregator never stores reports, so there is no report database to
// »   subpoena, leak or misconfigure. A design that aggregated later would need
// »   one.
//
// » WHAT THE "V" BUYS. Plain secret sharing hides the input — but a malicious
// » client could submit a share of 10^9 into what is supposed to be a 0/1 count
// » and poison the aggregate arbitrarily, invisibly, and repeatedly. The proof
// » makes that detectable WITHOUT revealing the input. Robustness against
// » malicious clients is not a bonus feature here; on a fleet of a billion
// » devices, an unverified aggregation protocol is a denial-of-integrity service
// » waiting to happen. This is the answer to "why does Prio need verifiability?"
//
// » READ, in this order:
// »   Corrigan-Gibbs & Boneh, "Prio" (NSDI 2017) — the original, very readable
// »   draft-irtf-cfrg-vdaf §1–5 (framework), §7 (Prio3), Appendix C (vectors)
// »   ISRG's libprio-rs and the janus aggregator — READ for shape, do not copy;
// »     writing it in Go teaches ten times more and is a differentiator against
// »     a Rust-heavy field
package prio3

import (
	"errors"

	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/field"
	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/xof"
)

// ErrTODO marks an unimplemented scaffold function.
var ErrTODO = errors.New("prio3: not implemented«, see the EXERCISE block above this function»")

// ErrInvalidReport is returned when the verification circuit rejects a report.
//
// » Distinguish this from a decoding or transport error and count them
// » separately in the metrics. A rising rate of invalid reports is a signal
// » about clients — a buggy rollout, or an attacker probing — while a rising
// » rate of decode errors is a signal about your own version skew. Collapsing
// » them into one counter destroys the only evidence you would have.
var ErrInvalidReport = errors.New("prio3: report failed verification")

// Shares is the number of aggregators. The draft supports more; two is what DAP
// deploys and what this implementation targets.
//
// » Why exactly two, when the maths generalises? Because the trust assumption is
// » "at least one aggregator is honest", and every additional aggregator is
// » another organisation to run, contract with and coordinate on key rotation.
// » Two is the smallest number that gives a meaningful non-collusion assumption,
// » and organisational cost — not cryptographic cost — is the binding
// » constraint. Divvi Up (ISRG + a partner) is the canonical example.
const Shares = 2

// LeaderID and HelperID index the aggregators.
const (
	LeaderID = 0
	HelperID = 1
)

// VDAF is the interface every Prio3 variant implements. The shape follows the
// draft's §5 so that the aggregator code in internal/dap can drive any variant
// without knowing which one it has.
type VDAF interface {
	// Name is the draft's identifier, e.g. "Prio3Count".
	Name() string

	// AlgorithmID is the registered VDAF algorithm ID, used in the wire format
	// and in domain separation.
	AlgorithmID() uint32

	// Shard splits a measurement into one public share and Shares input
	// shares. rand supplies the client's randomness; nonce is the report nonce.
	Shard(measurement any, nonce, rand []byte) (publicShare []byte, inputShares [][]byte, err error)

	// PrepInit begins verification for one aggregator.
	PrepInit(verifyKey []byte, aggID int, nonce, publicShare, inputShare []byte) (PrepState, []byte, error)

	// PrepSharesToPrep combines the aggregators' prep shares into the single
	// broadcast message that decides validity.
	PrepSharesToPrep(prepShares [][]byte) ([]byte, error)

	// PrepNext finishes verification, returning the output share to aggregate,
	// or ErrInvalidReport.
	PrepNext(state PrepState, prepMsg []byte) (OutputShare, error)

	// AggregateInit returns a zero aggregate share.
	AggregateInit() AggShare

	// Unshard combines one aggregate share per aggregator into the result.
	// numMeasurements is the batch size, needed by variants whose result
	// depends on it.
	Unshard(aggShares []AggShare, numMeasurements int) (any, error)
}

// PrepState is a variant's opaque per-report state between PrepInit and
// PrepNext.
//
// » Sizing this matters operationally: the leader holds one PrepState per
// » in-flight report for a full network round trip to the helper. At 10^5
// » reports/second and a 50 ms round trip that is 5,000 concurrent states; if
// » each is 8 KiB you have just found where your memory goes. The bounded queue
// » in internal/pipeline exists for exactly this class of problem, and the
// » aggregation job driver in internal/dap/leader.go needs the same discipline.
type PrepState interface {
	// Size reports the in-memory footprint in bytes, for capacity planning.
	Size() int
}

// OutputShare is one aggregator's verified share of one report's contribution.
type OutputShare []field.Field64

// AggShare is one aggregator's running sum of output shares over a batch.
type AggShare []field.Field64

// AddInto accumulates o into a, element-wise in the field.
//
// TODO(week2): implement.
func (a AggShare) AddInto(o OutputShare) error {
	// EXERCISE-BEGIN
	// ─── EXERCISE 18: the aggregation step ──────────────────────────────────
	// Task: element-wise field.Add, rejecting a length mismatch.
	//
	// Small function, big idea: THIS is why the protocol scales. The aggregator
	// keeps one vector of len(AggShare) field elements per batch, no matter how
	// many reports arrive — O(1) memory in batch size, and the sum can be
	// computed incrementally as reports stream in. Compare with a design that
	// stores reports to aggregate later: that one needs O(n) storage, and it
	// also means the reports exist on disk somewhere, which is a privacy
	// property you just gave away for no reason.
	//
	// Write two sentences in docs/PRIVACY.md about that trade-off. It is a good
	// interview answer to "how would you scale this".
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return ErrTODO
}

// Encode serialises the aggregate share for transport to the collector.
//
// TODO(week2): implement — field.Encode over each element.
func (a AggShare) Encode() []byte { return nil }

// DecodeAggShare parses an aggregate share of the expected length.
//
// TODO(week2): implement.
func DecodeAggShare(b []byte, n int) (AggShare, error) { return nil, ErrTODO }

// ─── the sharding core, shared by every variant ──────────────────────────────

// shardEncoded splits an already-encoded measurement vector into Shares input
// shares, using the seed trick so that only the leader's share travels in full.
//
// TODO(week2): implement.
func shardEncoded(x xof.XOF, encoded []field.Field64, seeds []xof.Seed, nonce []byte) ([][]field.Field64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 19: additive sharing with the seed trick ──────────────────
	// This is the function that makes the bandwidth numbers work. Build it in
	// two stages and keep both, because stage 1 is your oracle for stage 2.
	//
	// STAGE 1 (naive, correct, easy to debug): for each aggregator except the
	//   last, draw a uniformly random vector rᵢ. The last share is
	//   encoded − Σrᵢ. Verify that the shares sum to `encoded` element-wise.
	//   Note this must use xof.NextField64's rejection sampling, NOT modular
	//   reduction of raw bytes — see EXERCISE 16 for why the bias matters here
	//   specifically: "the mask is uniform" is the whole security argument.
	//
	// STAGE 2 (the real one): the helper's share is not sent. Instead:
	//   - the client picks a 16-byte seed and expands it with the XOF into the
	//     helper's whole share vector;
	//   - the leader's share is encoded − expand(seed);
	//   - the report carries the leader's full vector plus the helper's 16-byte
	//     seed. The helper re-expands it locally.
	//   Domain separation and the binder must match the draft exactly or the
	//   helper derives a different vector and the aggregate is silently wrong.
	//
	// MEASURE IT (this is a deliverable, not a nicety): report bytes for
	//   Prio3Histogram with 1024 buckets, stage 1 vs stage 2, in
	//   docs/BENCHMARKS.md. You should see roughly 2× → 1× + 16 bytes.
	//
	// THEN THE QUESTION THAT MAKES IT INTERESTING: the seed trick is asymmetric
	//   — the leader receives ~8 KiB and the helper ~16 bytes. What does that
	//   asymmetry do to the cost of running each role, and does it change who is
	//   willing to be the helper? Real deployments care a great deal about this,
	//   and it is a genuinely good thing to have an opinion about.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// jointRandomness derives the random challenge point the FLP is verified at,
// binding it to all input shares so that no single party controls it.
//
// TODO(week2): implement.
func jointRandomness(x xof.XOF, seeds []xof.Seed, nonce []byte, n int) ([]field.Field64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 20: joint randomness, and why it is joint ─────────────────
	// The FLP is checked at a random point. Who picks it?
	//
	//   - If the CLIENT picks it, a malicious client picks a point where its
	//     bogus proof happens to verify, and robustness is gone.
	//   - If ONE AGGREGATOR picks it alone, a malicious aggregator picks a point
	//     that makes an honest client's valid report fail — a targeted
	//     denial-of-service against specific users, which is a privacy problem
	//     as well as an availability one.
	//
	// So it is derived from a commitment to BOTH input shares plus the nonce:
	// each aggregator computes a "blind" over its own share, they exchange
	// those, and the challenge is derived from the pair. Nobody controls it, and
	// each side can check the other used the real one.
	//
	// Task: implement per VDAF draft §7.2.2 (joint randomness derivation).
	//       Read it closely — the draft's two-step structure (part → seed →
	//       expanded randomness) exists to keep the per-report overhead at one
	//       seed rather than one full vector.
	// Test: flip one byte of an input share and assert the derived randomness
	//       changes. Then flip one byte of the NONCE and assert the same. The
	//       second case is what stops a report being replayed across tasks.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// EXERCISE-BEGIN
// ─── EXERCISE 21 (week 3, after the vectors pass): Poplar1, or not ───────────
// The other VDAF in the draft is Poplar1, which solves HEAVY HITTERS: "what are
// the 100 most common values", where the value space is far too large to
// enumerate as histogram buckets (URLs, crash signatures, GPU kernel names —
// the last one is directly relevant to gpu-trace).
//
// It works by prefix-counting over several rounds: count how many clients have
// each 1-bit prefix, keep the popular ones, then ask about 2-bit prefixes, and
// so on. The primitive underneath is an incremental distributed point function.
//
// Task (choose one, honestly):
//   (a) IMPLEMENT it. Genuinely hard — IDPFs are a step up in difficulty from
//       Prio3 and will cost you most of a week.
//   (b) UNDERSTAND it. Read the Poplar paper and draft §8, then write half a
//       page in docs/PRIVACY.md: what problem it solves, why a histogram cannot,
//       what the multi-round structure costs operationally (the aggregators must
//       keep per-round state across collections, which is a much heavier
//       operational commitment than Prio3's stateless streaming aggregation),
//       and when you would reach for it in a gpu-trace context.
//
// Recommendation: (b). The plan lists Poplar1 as the second thing to drop if
// you fall behind, and a crisp paragraph about a mechanism you chose not to
// build reads better in an interview than a half-finished IDPF. Being able to
// say "I read it, here is the trade-off, I decided against it" is a senior
// answer.
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END
