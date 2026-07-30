package prio3

import (
	"fmt"

	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/field"
	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/xof"
)

// » THE ONE THIS PROJECT ACTUALLY NEEDS. gpu-trace produces continuous
// » quantities — SM utilisation, memory occupancy, kernel duration. A histogram
// » over bucketed values is the natural private summary: "the distribution of
// » GPU utilisation across the fleet in this five-minute window", with no
// » per-device value ever visible to anyone.
// »
// » Build Prio3Count first (it isolates the sharding machinery), then this.

// Prio3Histogram counts clients into Length disjoint buckets. The measurement
// is an int bucket index; the aggregate is []uint64 of length Length.
//
// » TWO constraints, and both are load-bearing:
// »   1. every entry is a bit:        xᵢ² − xᵢ = 0 for all i
// »   2. the entries sum to one:      Σxᵢ = 1
// »
// » Constraint 2 is the one implementations forget, and forgetting it is
// » exploitable: without it a client submits all-ones and adds 1 to EVERY bucket,
// » or submits all-zeros and opts out of a mandatory metric undetectably. With it,
// » a client contributes to exactly one bucket, which is what makes the
// » sensitivity analysis in internal/dp/query.go (Histogram, MaxBuckets = 1)
// » actually true rather than merely assumed. Note the chain: a missing circuit
// » constraint invalidates a DP parameter three packages away. That coupling
// » between the crypto and the privacy accounting is the thing to be able to
// » explain out loud.
// »
// » Combining the two into ONE check needs joint randomness: the aggregators
// » take a random linear combination of all Length + 1 constraints, so a single
// » verifier comparison covers all of them. That is why JointRandLen is non-zero
// » here and zero for Count.
func Prio3Histogram(length, chunkLength int, newXOF func() xof.XOF) (VDAF, error) {
	if length <= 0 {
		return nil, fmt.Errorf("prio3: histogram length must be > 0, got %d", length)
	}
	if chunkLength <= 0 {
		return nil, fmt.Errorf("prio3: chunk length must be > 0, got %d", chunkLength)
	}
	return &driver{
		name:  "Prio3Histogram",
		algID: AlgorithmIDHistogram,
		flp:   &histogramFLP{length: length, chunkLength: chunkLength},
		newX:  newXOF,
	}, nil
}

// OptimalChunkLength returns a reasonable chunkLength for a histogram of the
// given length.
//
// » WHAT CHUNK LENGTH IS, because it is the least obvious parameter in the whole
// » draft and it is the one you will be asked to justify. The bit-check
// » constraints are batched into chunks and each chunk becomes one call to a
// » ParallelSum gadget. It trades proof size against the number of gadget calls:
// »
// »   small chunks → many gadget calls → longer proof, more client CPU
// »   large chunks → fewer calls → shorter proof, but higher gadget degree,
// »                  which pushes proof size back up
// »
// » The optimum is near √Length. The draft gives a concrete procedure (§7.4.3);
// » implement that rather than this square-root approximation once the vectors
// » matter, because the test vectors pin a specific value.
// »
// » Then measure it: plot report size against chunkLength for Length = 1024 and
// » put it in docs/BENCHMARKS.md. "I tuned a protocol parameter against measured
// » bytes-on-wire" is a much better sentence than "I used the default."
//
// TODO(week2): implement the draft's procedure.
func OptimalChunkLength(length int) int {
	return 0
}

// histogramFLP is the Prio3Histogram validity circuit.
type histogramFLP struct {
	length      int
	chunkLength int
}

// InputLen implements FLP.
func (h *histogramFLP) InputLen() int { return h.length }

// OutputLen implements FLP.
func (h *histogramFLP) OutputLen() int { return h.length }

// JointRandLen implements FLP.
//
// TODO(week2): implement — how many random coefficients does the linear
// combination of the Length + 1 constraints need? Read draft §7.4.
func (h *histogramFLP) JointRandLen() int { return 0 }

// ProofLen implements FLP.
//
// TODO(week2): implement — derive from the gadget, do not hardcode.
func (h *histogramFLP) ProofLen() int { return 0 }

// VerifierLen implements FLP.
//
// TODO(week2): implement.
func (h *histogramFLP) VerifierLen() int { return 0 }

// ProveRandLen implements FLP.
//
// TODO(week2): implement.
func (h *histogramFLP) ProveRandLen() int { return 0 }

// QueryRandLen implements FLP.
//
// TODO(week2): implement.
func (h *histogramFLP) QueryRandLen() int { return 0 }

// Encode implements FLP: bucket index → one-hot vector.
//
// TODO(week2): implement.
func (h *histogramFLP) Encode(measurement any) ([]field.Field64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 29: one-hot encoding, and the cost of bucket count ─────────
	// Task: assert measurement to int, reject anything outside [0, length), and
	//       return a vector with a single 1.
	//
	// Notice the bandwidth: a 1024-bucket histogram is a 1024-element input
	// vector, 8 KiB, to convey ONE number in [0, 1024) — ten bits of information
	// in eight kilobytes. That is not waste, it is the price of the privacy
	// property: the aggregators must not learn which bucket, so the client has
	// to touch all of them.
	//
	// Which is exactly why the seed trick in EXERCISE 19 matters so much, and
	// exactly why bucket count is a privacy-and-bandwidth decision rather than a
	// UX one. Put the numbers in docs/BENCHMARKS.md:
	//
	//   buckets | plaintext OTLP bytes | Prio3 report bytes | ratio
	//        16 |                      |                    |
	//       128 |                      |                    |
	//      1024 |                      |                    |
	//
	// That table IS "quantify the privacy tax" from the plan. A hiring manager on
	// a billions-of-devices team will go to it first.
	//
	// Then answer, because it is the natural follow-up: for a continuous quantity
	// like SM utilisation, would you rather have 1024 linear buckets or 64
	// logarithmic ones? (Consider where the interesting mass of a utilisation
	// distribution actually is, and what DP noise does to a bucket whose true
	// count is 3.)
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// Truncate implements FLP: the one-hot vector is the output.
//
// TODO(week2): implement.
func (h *histogramFLP) Truncate(input []field.Field64) ([]field.Field64, error) {
	return nil, ErrTODO
}

// Decode implements FLP: field elements → []uint64 bucket counts.
//
// TODO(week2): implement.
func (h *histogramFLP) Decode(agg []field.Field64, numMeasurements int) (any, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 30: the wraparound question ───────────────────────────────
	// Each bucket count is a field element. The field is GF(2^64 − 2^32 + 1), so
	// a count of 5 and a count of 5 + p are the same element.
	//
	// Task A: decide what Decode does with an element that cannot be a plausible
	//         count — larger than numMeasurements, say. Reject, or clamp? Argue
	//         it, and write the argument in docs/PRIVACY.md rather than just
	//         picking one.
	// Task B: then work out whether it is reachable. Can a VERIFIED batch produce
	//         an element above numMeasurements? (Consider what the circuit
	//         guarantees per report, and what summing n verified reports can and
	//         cannot produce.) Getting to a confident "no, and here is why" is
	//         the useful outcome — but keep the check anyway, as a
	//         defence-in-depth assertion that would fire loudly if the circuit
	//         were ever weakened by a refactor.
	// Task C: now add DP noise on top (internal/dap/collector.go). Noise is
	//         signed. A true count of 2 plus noise of −5 is −3, which in the
	//         field is p − 3, which as a uint64 is astronomical. Decide where the
	//         signed-to-unsigned conversion happens and what you publish for a
	//         negative noisy count: clamp to zero, or report the negative value?
	//         There is a real answer and it is NOT "clamp" — clamping introduces
	//         a bias that breaks downstream statistics, and post-processing a DP
	//         output is free privacy-wise, so consumers can clamp at display
	//         time if they want. Publishing signed values and documenting them is
	//         the defensible choice. Say which you chose and why.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// Prove implements FLP.
//
// TODO(week2): implement — draft §7.4, ParallelSum over the bit-check chunks.
func (h *histogramFLP) Prove(input, proveRand, jointRand []field.Field64) ([]field.Field64, error) {
	return nil, ErrTODO
}

// Query implements FLP.
//
// TODO(week2): implement — draft §7.4.
func (h *histogramFLP) Query(inputShare, proofShare, queryRand, jointRand []field.Field64, shares int) ([]field.Field64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 31: the "shares" argument, and why it is there ────────────
	// Query takes the number of shares. Look at the sum-to-one constraint and
	// work out why.
	//
	// The constraint is Σxᵢ − 1 = 0. Each aggregator evaluates it on its own
	// share. But the constant −1 is NOT secret-shared: if both aggregators
	// subtract 1, the parts sum to Σxᵢ − 2, and a valid report fails. The
	// constant must be split across the shares, so each subtracts 1/shares.
	//
	// Task: get this right, and put a test on it that runs with shares = 2 and
	//       would catch a hardcoded 1. Every constant term in every constraint
	//       has this problem, so make it a rule you apply automatically: in a
	//       shared computation, constants are divided among the shares.
	//
	// This is a small, sharp piece of understanding about secret-shared
	// computation, it is the kind of thing an interviewer can probe in two
	// minutes, and having hit it yourself is very different from having read it.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// Decide implements FLP.
//
// TODO(week2): implement« — see EXERCISE 28».
func (h *histogramFLP) Decide(verifier []field.Field64) (bool, error) {
	return false, ErrTODO
}
