package prio3

import (
	"fmt"

	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/field"
	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/xof"
)

// » Prio3Sum answers "total GPU-seconds consumed across the fleet" without
// » learning any device's contribution. Build it after Count and Histogram; the
// » interesting new idea is bit decomposition.

// Prio3Sum sums per-client integers in [0, MaxMeasurement]. The measurement is
// a uint64; the aggregate is a uint64.
//
// » THE PROBLEM THIS SOLVES, and it is the whole reason Sum is not trivial.
// » Nothing in the field stops a client submitting p − 1 as its "utilisation".
// » The field has no order, so there is no polynomial identity for "x ≤ 1000"
// » the way there is for "x ∈ {0,1}".
// »
// » THE TRICK: make the client send x in BINARY, as ⌈log₂(max+1)⌉ separate
// » field elements, prove each one is a bit (which you already know how to do —
// » it is Prio3Count's constraint, repeated), and reconstruct
// » x = Σ 2^i · bᵢ, which is LINEAR and therefore free for the aggregators. A
// » number that is a sum of k bits is automatically in [0, 2^k), so the range
// » constraint comes for free from the encoding rather than being checked.
// »
// » Encoding a value so that the property you need is structurally impossible to
// » violate, instead of checking it afterwards, is a genuinely transferable
// » idea. It is the same instinct as making an illegal state unrepresentable in
// » a type system.
// »
// » THE COST: a 32-bit range means 32 field elements — 256 bytes — for one
// » number. Compare that with 8 bytes plaintext. That 32× expansion is the
// » privacy tax on a sum, and Prio3SumVec amortises it across many sums in one
// » report, which is why real deployments use SumVec rather than many Sums. Note
// » that in docs/BENCHMARKS.md.
func Prio3Sum(maxMeasurement uint64, newXOF func() xof.XOF) (VDAF, error) {
	if maxMeasurement == 0 {
		return nil, fmt.Errorf("prio3: maxMeasurement must be > 0")
	}
	return &driver{
		name:  "Prio3Sum",
		algID: AlgorithmIDSum,
		flp:   &sumFLP{max: maxMeasurement},
		newX:  newXOF,
	}, nil
}

// sumFLP is the Prio3Sum validity circuit.
type sumFLP struct {
	max uint64
	// bits is ⌈log₂(max + 1)⌉, the length of the bit decomposition.
	bits int
}

// Bits returns the number of bits in the decomposition.
//
// TODO(week2): implement — and watch the boundary: max = 1 needs 1 bit, max = 2
// needs 2, max = 255 needs 8, max = 256 needs 9. Off-by-one here means every
// report is a byte too small or too large and the vectors fail.
func (s *sumFLP) Bits() int { return s.bits }

// InputLen implements FLP.
//
// TODO(week2): implement.
func (s *sumFLP) InputLen() int { return 0 }

// OutputLen implements FLP.
//
// » One: the sum is a single number, even though the input is `bits` elements.
// » This is the only variant so far where InputLen and OutputLen differ, which
// » is precisely what Truncate exists for — the aggregators recombine the bits
// » into the value before aggregating, so the running aggregate is one element
// » rather than `bits` of them.
func (s *sumFLP) OutputLen() int { return 1 }

// JointRandLen implements FLP.
//
// TODO(week2): implement — the bit checks are combined randomly, as in the
// histogram.
func (s *sumFLP) JointRandLen() int { return 0 }

// ProofLen implements FLP.
//
// TODO(week2): implement.
func (s *sumFLP) ProofLen() int { return 0 }

// VerifierLen implements FLP.
//
// TODO(week2): implement.
func (s *sumFLP) VerifierLen() int { return 0 }

// ProveRandLen implements FLP.
//
// TODO(week2): implement.
func (s *sumFLP) ProveRandLen() int { return 0 }

// QueryRandLen implements FLP.
//
// TODO(week2): implement.
func (s *sumFLP) QueryRandLen() int { return 0 }

// Encode implements FLP: uint64 → little-endian bit vector.
//
// TODO(week2): implement.
func (s *sumFLP) Encode(measurement any) ([]field.Field64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 32: bit decomposition ─────────────────────────────────────
	// Task: assert to uint64, reject > s.max, and return s.bits field elements,
	//       LEAST significant bit first (check the draft's endianness — VDAF §7.5
	//       — and do not guess; the field encoding is little-endian and it is
	//       tempting to assume the bit order matches, which is a separate
	//       decision).
	//
	// The subtle one, and it is the interesting part of this exercise: rejecting
	// max < x < 2^bits. With max = 1000, bits = 10, so the bit decomposition
	// permits values up to 1023. A client submitting 1010 passes every bit check
	// and exceeds your declared bound — which quietly invalidates the
	// sensitivity you claimed in internal/dp/query.go. The draft handles this
	// with an extra offset check; read §7.5 and implement it. Then write the
	// negative test.
	//
	// Sit with the shape of that bug for a moment, because it recurs: the
	// crypto was sound, every constraint verified, and the PRIVACY GUARANTEE was
	// still wrong, because the contribution bound the DP analysis assumed was not
	// the bound the circuit enforced. Most real failures in this field look like
	// that — a correct component composed under a false assumption — not like a
	// broken cipher.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// Truncate implements FLP: recombine the bits into the value.
//
// TODO(week2): implement — Σ 2^i · bᵢ, a linear function and therefore free.
func (s *sumFLP) Truncate(input []field.Field64) ([]field.Field64, error) {
	return nil, ErrTODO
}

// Decode implements FLP.
//
// TODO(week2): implement.
func (s *sumFLP) Decode(agg []field.Field64, numMeasurements int) (any, error) {
	return nil, ErrTODO
}

// Prove implements FLP.
//
// TODO(week2): implement — draft §7.5.
func (s *sumFLP) Prove(input, proveRand, jointRand []field.Field64) ([]field.Field64, error) {
	return nil, ErrTODO
}

// Query implements FLP.
//
// TODO(week2): implement — draft §7.5, and mind the shares-divided constants
// from« EXERCISE 31».
func (s *sumFLP) Query(inputShare, proofShare, queryRand, jointRand []field.Field64, shares int) ([]field.Field64, error) {
	return nil, ErrTODO
}

// Decide implements FLP.
//
// TODO(week2): implement« — see EXERCISE 28».
func (s *sumFLP) Decide(verifier []field.Field64) (bool, error) {
	return false, ErrTODO
}

// EXERCISE-BEGIN
// ─── EXERCISE 33 (stretch, high value): Prio3SumVec ─────────────────────────
// SumVec sums a VECTOR of bounded integers in one report, amortising the proof
// across all of them. It is the variant real deployments use, because one report
// carrying 50 metrics is far cheaper than 50 reports carrying one each — one
// proof, one HPKE encryption, one HTTP request, one anti-replay entry.
//
// Task: implement it (draft §7.6). It is mostly Prio3Sum with an outer loop, so
//       the marginal cost after Sum is small and the marginal credibility is
//       high: it is what you would actually deploy for gpu-trace, where a device
//       reports SM utilisation, memory occupancy, temperature and clock together.
// Then: measure report size for 50 metrics as 50 Prio3Sums versus one SumVec,
//       and put the ratio in docs/BENCHMARKS.md. That single number is a good
//       answer to "how would you make this cheap enough for a phone."
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END
