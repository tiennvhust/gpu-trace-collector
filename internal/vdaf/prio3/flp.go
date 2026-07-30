package prio3

import "github.com/tiennvhust/gpu-trace-collector/internal/vdaf/field"

// » FULLY LINEAR PROOFS, the one genuinely new idea in this project. Read this
// » comment twice before writing code; the API below only makes sense afterwards.
// »
// » THE PROBLEM. The aggregators hold additive shares of a vector x. They need
// » to decide whether x satisfies a validity predicate — "every entry is 0 or 1
// » and exactly one is 1", say — without learning x, and without any expensive
// » multi-party computation, because this has to run on billions of reports.
// »
// » WHAT MAKES IT POSSIBLE. Shares are additive, so the aggregators can compute
// » any LINEAR function of x for free: each evaluates the linear function on its
// » own share and the results add up to the function of x. Linear is free;
// » multiplication is not.
// »
// » THE TRICK. Express validity as an arithmetic circuit C with C(x) = 0 exactly
// » when x is valid. The circuit needs multiplications — "b is a bit" is
// » b² − b = 0 — so the CLIENT does them in advance and ships the intermediate
// » products as a proof π. Now the aggregators only have to check consistency
// » between x and π, and that check can be written as a LINEAR function of
// » (x, π) once a random challenge point is fixed. Linear, so it is free.
// »
// » WHY A CHEATING CLIENT CANNOT WIN. If x is invalid, the polynomial identity
// » the proof asserts is a non-zero polynomial. A non-zero polynomial of degree d
// » has at most d roots in a field of size p, so a random challenge catches the
// » lie with probability at least 1 − d/p. With p ≈ 2^64 and d small, that is
// » about 1 − 2^−60. Schwartz–Zippel; one line, and it is the whole soundness
// » argument.
// »
// » WHY IT REVEALS NOTHING. The proof is itself secret-shared, and the check
// » output is compared against zero. An honest client's report makes each
// » aggregator's view a uniformly random value subject only to the two summing
// » to zero — one bit of information total, "valid or not". That is the privacy
// » argument, and "one bit leaks" is the honest thing to write in
// » THREAT_MODEL.md rather than "nothing leaks".
// »
// » Spec: VDAF draft §7.1 (FLP), §7.3–7.5 (the three circuits below).
// » Paper: Boneh, Boyle, Corrigan-Gibbs, Gilboa, Ishai, "Zero-Knowledge Proofs
// » on Secret-Shared Data via Fully Linear PCPs" (CRYPTO 2019).

// FLP is the validity circuit for one Prio3 variant.
//
// » The five sizes below are the whole interface contract, and getting one of
// » them wrong is the most common cause of a test-vector mismatch that reports
// » no error anywhere — the shares just stop adding up.
type FLP interface {
	// Encode turns a measurement into the input vector the circuit validates.
	Encode(measurement any) ([]field.Field64, error)

	// Truncate maps a valid input vector to the values that get aggregated.
	// For a histogram they are the same; for a sum they are not.
	Truncate(input []field.Field64) ([]field.Field64, error)

	// Decode turns an aggregated, truncated vector back into a result.
	Decode(agg []field.Field64, numMeasurements int) (any, error)

	// Prove generates the proof for input at the given prover randomness.
	Prove(input, proveRand, jointRand []field.Field64) ([]field.Field64, error)

	// Query evaluates the verifier's linear function on one SHARE of the input
	// and proof. The results across aggregators sum to the verifier message.
	Query(inputShare, proofShare, queryRand, jointRand []field.Field64, shares int) ([]field.Field64, error)

	// Decide returns whether the combined verifier message accepts.
	Decide(verifier []field.Field64) (bool, error)

	// InputLen is the length of the encoded input vector.
	InputLen() int
	// ProofLen is the length of the proof.
	ProofLen() int
	// VerifierLen is the length of the verifier message.
	VerifierLen() int
	// OutputLen is the length of the truncated output.
	OutputLen() int
	// JointRandLen is how many joint-randomness elements the circuit consumes.
	JointRandLen() int
	// ProveRandLen is how many prover-randomness elements Prove consumes.
	ProveRandLen() int
	// QueryRandLen is how many query-randomness elements Query consumes.
	QueryRandLen() int
}

// ─── gadgets ─────────────────────────────────────────────────────────────────

// Gadget is a sub-circuit the FLP calls a bounded number of times. Gadgets are
// where all the multiplication lives; everything outside them is linear and
// therefore free for the aggregators.
//
// » The FLP machinery cares about exactly two things per gadget: its arity and
// » its degree. Arity fixes how many wires feed it; degree fixes the degree of
// » the polynomial the proof must encode, which fixes ProofLen, which fixes
// » report size. If you find yourself wanting a higher-degree gadget, you are
// » proposing to make every device's upload bigger — worth knowing that is the
// » trade you are making.
type Gadget interface {
	// Eval applies the gadget to one set of inputs.
	Eval(in []field.Field64) (field.Field64, error)

	// EvalPoly applies the gadget to polynomials, for proof generation.
	EvalPoly(in []field.Poly) (field.Poly, error)

	// Arity is the number of inputs.
	Arity() int
	// Degree is the polynomial degree of the gadget.
	Degree() int
	// Calls is how many times the circuit invokes this gadget.
	Calls() int
}

// MulGadget multiplies two field elements. Arity 2, degree 2.
type MulGadget struct{ NumCalls int }

// Arity implements Gadget.
func (g *MulGadget) Arity() int { return 2 }

// Degree implements Gadget.
func (g *MulGadget) Degree() int { return 2 }

// Calls implements Gadget.
func (g *MulGadget) Calls() int { return g.NumCalls }

// Eval implements Gadget.
//
// TODO(week2): implement — field.Mul of the two inputs, with an arity check.
func (g *MulGadget) Eval(in []field.Field64) (field.Field64, error) {
	return 0, ErrTODO
}

// EvalPoly implements Gadget.
//
// TODO(week2): implement — polynomial multiplication«, see EXERCISE 22».
func (g *MulGadget) EvalPoly(in []field.Poly) (field.Poly, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 22: the gadget polynomial ─────────────────────────────────
	// Task: return in[0] · in[1] using field.Poly.Mul.
	//
	// The idea to hold on to, because it is what the proof IS: the prover
	// interpolates a polynomial through the gadget's inputs across all of its
	// calls, so that evaluating that polynomial at the k-th canonical point
	// reproduces the k-th call's input. The gadget applied to those input
	// polynomials gives an OUTPUT polynomial whose evaluations at the canonical
	// points are exactly the gadget's outputs. The proof is that polynomial's
	// coefficients. Checking it at one random point therefore checks every call
	// at once — which is why the proof is short and constant-ish rather than
	// proportional to the number of multiplications.
	// Spec: VDAF draft §7.1.2 and §7.1.3.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// PolyEvalGadget applies a fixed univariate polynomial. Arity 1, degree
// len(Coeffs) − 1. Prio3Histogram and Prio3Sum both use one.
type PolyEvalGadget struct {
	Coeffs   field.Poly
	NumCalls int
}

// Arity implements Gadget.
func (g *PolyEvalGadget) Arity() int { return 1 }

// Degree implements Gadget.
func (g *PolyEvalGadget) Degree() int { return g.Coeffs.Degree() }

// Calls implements Gadget.
func (g *PolyEvalGadget) Calls() int { return g.NumCalls }

// Eval implements Gadget.
//
// TODO(week2): implement — g.Coeffs.Eval(in[0]).
func (g *PolyEvalGadget) Eval(in []field.Field64) (field.Field64, error) {
	return 0, ErrTODO
}

// EvalPoly implements Gadget.
//
// TODO(week2): implement — compose g.Coeffs with in[0].
func (g *PolyEvalGadget) EvalPoly(in []field.Poly) (field.Poly, error) {
	return nil, ErrTODO
}

// EXERCISE-BEGIN
// ─── EXERCISE 23: prove the proof actually catches cheating ──────────────────
// A verifier that accepts everything passes every happy-path test you will
// write. So write the negative tests deliberately, in flp_test.go:
//
//   1. VALID input → Prove → Query on both shares → Decide == true.
//   2. Flip one element of the input vector, keep the original proof → false.
//   3. Keep a valid input, corrupt one proof element → false.
//   4. Prio3Count with measurement 2 instead of 0 or 1 → Encode should reject,
//      and if you hand-craft the vector past Encode, Decide → false.
//   5. Prio3Histogram with TWO buckets set to 1 → false. (This is the check
//      people forget: "each entry is a bit" and "the entries sum to one" are two
//      separate constraints, and an implementation missing the second one is
//      exploitable — a client stuffs every bucket and skews the whole metric.)
//   6. Prio3Sum with a value above the configured bound → false.
//
// Then measure the soundness error empirically: run case 2 with 10,000 random
// challenge points and confirm zero acceptances. You are checking d/p ≈ 2^−60,
// so zero is the only acceptable count, and any acceptance at all means the
// challenge is not actually random — which points back at EXERCISE 20.
//
// These six tests are what turn "I implemented Prio3" into "I verified Prio3's
// robustness property". That distinction is the interview answer.
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END
