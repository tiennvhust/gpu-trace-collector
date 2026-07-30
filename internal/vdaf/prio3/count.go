package prio3

import (
	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/field"
	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/xof"
)

// » BUILD THIS ONE FIRST. Prio3Count is the smallest complete Prio3: input
// » vector of length 1, one gadget call, one validity constraint. Every bug you
// » will hit in Prio3Sum and Prio3Histogram is present here in miniature and is
// » ten times easier to find. Do not skip ahead to the histogram because it is
// » the one the project actually needs.
// »
// » In gpu-trace terms: "how many devices saw a CUDA OOM in this window."

// Prio3Count counts the number of clients reporting 1. The measurement is a
// bool; the aggregate is a uint64.
//
// » THE ENTIRE VALIDITY CIRCUIT IS ONE EQUATION:
// »
// »     x · x − x = 0   ⟺   x ∈ {0, 1}
// »
// » One MulGadget call. Sit with how little that is: the client proves, to two
// » mutually distrustful parties who never see its value, that its value was a
// » bit — and the proof is a couple of field elements. When people say
// » "verifiable aggregation is cheap", this is what they mean.
func Prio3Count(newXOF func() xof.XOF) VDAF {
	return &driver{
		name:  "Prio3Count",
		algID: AlgorithmIDCount,
		flp:   &countFLP{},
		newX:  newXOF,
	}
}

// Registered VDAF algorithm IDs (VDAF draft §10, the "VDAF Identifiers"
// registry).
//
// » Look these up in the current draft rather than trusting the values here —
// » they are part of the wire format and they moved between draft versions. The
// » algorithm ID feeds into domain separation, so a wrong value fails the test
// » vectors with, again, no error message.
const (
	AlgorithmIDCount     uint32 = 0x00000000
	AlgorithmIDSum       uint32 = 0x00000001
	AlgorithmIDSumVec    uint32 = 0x00000002
	AlgorithmIDHistogram uint32 = 0x00000003
)

// countFLP is the Prio3Count validity circuit.
type countFLP struct{}

// InputLen implements FLP.
func (c *countFLP) InputLen() int { return 1 }

// OutputLen implements FLP.
func (c *countFLP) OutputLen() int { return 1 }

// JointRandLen implements FLP.
//
// » Zero, and this is worth pausing on. The circuit has a single constraint, so
// » there is nothing for a random linear combination to combine — no joint
// » randomness is needed at all. That in turn means Prio3Count skips the
// » joint-randomness exchange in« EXERCISE 20» entirely, which is a second reason
// » to implement Count first: it isolates the sharding and proof machinery from
// » the joint-randomness machinery so you debug them one at a time.
func (c *countFLP) JointRandLen() int { return 0 }

// ProofLen implements FLP.
//
// TODO(week2): implement — derive from the gadget's degree and call count per
// draft §7.1; do not hardcode a number you read off a test vector.
func (c *countFLP) ProofLen() int { return 0 }

// VerifierLen implements FLP.
//
// TODO(week2): implement.
func (c *countFLP) VerifierLen() int { return 0 }

// ProveRandLen implements FLP.
//
// TODO(week2): implement.
func (c *countFLP) ProveRandLen() int { return 0 }

// QueryRandLen implements FLP.
//
// TODO(week2): implement.
func (c *countFLP) QueryRandLen() int { return 0 }

// Encode implements FLP: bool → [0] or [1].
//
// TODO(week2): implement.
func (c *countFLP) Encode(measurement any) ([]field.Field64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 27: encode, and fail closed on the type ───────────────────
	// Task: type-assert measurement to bool and return field.Poly{0} or {1}.
	//       Return a clear error for any other type — including int(0) and
	//       int(1), which look harmless and are not: accepting int here means
	//       accepting int(2) tomorrow, and the whole point of Encode is that
	//       it is the chokepoint where an invalid measurement cannot get in.
	// Then read internal/privacy/encoder.go, which is where OTLP datapoints turn
	//       into measurements, and make sure the clamping happens THERE and not
	//       here — Encode should reject, not repair. A function that quietly
	//       repairs bad input is how contribution bounds stop being enforced.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// Truncate implements FLP: the input is already the output.
//
// TODO(week2): implement.
func (c *countFLP) Truncate(input []field.Field64) ([]field.Field64, error) {
	return nil, ErrTODO
}

// Decode implements FLP: one field element → uint64.
//
// TODO(week2): implement.
func (c *countFLP) Decode(agg []field.Field64, numMeasurements int) (any, error) {
	return nil, ErrTODO
}

// Prove implements FLP.
//
// TODO(week2): implement per draft §7.3«, see EXERCISE 22».
func (c *countFLP) Prove(input, proveRand, jointRand []field.Field64) ([]field.Field64, error) {
	return nil, ErrTODO
}

// Query implements FLP.
//
// TODO(week2): implement per draft §7.3«, see EXERCISE 25».
func (c *countFLP) Query(inputShare, proofShare, queryRand, jointRand []field.Field64, shares int) ([]field.Field64, error) {
	return nil, ErrTODO
}

// Decide implements FLP.
//
// TODO(week2): implement.
func (c *countFLP) Decide(verifier []field.Field64) (bool, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 28: compare in constant time, and mean it ─────────────────
	// The check is "is every element of the verifier zero". Use
	// crypto/subtle.ConstantTimeCompare against a zero vector rather than an
	// early-exiting loop.
	//
	// Be honest with yourself about whether it matters here, because "use
	// constant time" as a reflex without the argument is cargo cult. The
	// reasoning: an aggregator's rejection TIMING is observable to whoever is
	// watching the aggregation job, and if the timing depends on WHICH element
	// was non-zero, it leaks a little about the malformed input. Small, but the
	// habit is free and you are working in a codebase where the same reflex
	// (EXERCISE 1 in internal/tenant) already applies to API keys.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return false, ErrTODO
}
