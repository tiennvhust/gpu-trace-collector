// Package field implements arithmetic in the prime fields used by VDAF.
//
// » START HERE. This is the bottom of the Prio3 stack and everything above it
// » is plumbing. If the field is wrong, nothing else can be debugged — every
// » higher-level failure looks like "the shares don't add up" with no clue
// » attached. Get this package green before writing a single line of prio3.
//
// » WHY A FIELD AT ALL. Additive secret sharing needs a group where "add a
// » uniformly random value" perfectly hides the operand. Over the integers it
// » cannot: mask x with random r and x + r still leaks, because r must come from
// » some bounded range and the sum's distribution depends on x. Over Z_p with p
// » prime, x + r mod p for uniform r is EXACTLY uniform, independent of x. That
// » is a one-line proof and it is the foundation of the whole protocol:
// »
// »   Give aggregator A the share r, aggregator B the share x − r mod p. Each
// »   share alone is uniform, so each aggregator alone learns nothing whatever
// »   about x. Together they add to x. Sum a million reports share-wise and the
// »   two aggregate shares add to the true sum.
// »
// » Prio needs a field rather than just a group because verifying "the client's
// » input was well-formed" requires MULTIPLYING shared values, which is what the
// » FLP does. See internal/vdaf/prio3/flp.go.
//
// » Spec: draft-irtf-cfrg-vdaf, §6 (Fields). Read that section next to this file.
// »   https://datatracker.ietf.org/doc/draft-irtf-cfrg-vdaf/
package field

import (
	"errors"
	"math/big"
)

// ErrTODO marks an unimplemented scaffold function.
var ErrTODO = errors.New("field: not implemented«, see the EXERCISE block above this function»")

// Field64 is an element of GF(p) with p = 2^64 − 2^32 + 1, the field the VDAF
// draft calls Field64. Values are always kept reduced, in [0, p).
//
// » WHY THIS PRIME (and it is a lovely one). p = 2^64 − 2^32 + 1 is the
// » "Goldilocks" prime. Two properties earn it its place:
// »
// »   1. Reduction is cheap. Because 2^64 ≡ 2^32 − 1 (mod p), you can reduce a
// »      128-bit product with a handful of shifts, adds and one or two
// »      conditional subtractions — no division, no Montgomery form. Look at
// »      the identity for a moment before you implement Mul; deriving the
// »      reduction yourself is a satisfying twenty minutes and it is the reason
// »      this field is fast enough for billions of devices.
// »   2. p − 1 = 2^32 · 3 · 5 · 17 · 257 · 65537 is divisible by a large power
// »      of two, so the field has 2^32-th roots of unity. That is what makes the
// »      FFT-based polynomial interpolation in poly.go possible, which is what
// »      keeps proof generation near-linear instead of quadratic.
// »
// » Choosing a field whose modulus is convenient is not an optimisation detail
// » in this world, it is the design.
type Field64 uint64

// Modulus64 is p = 2^64 − 2^32 + 1.
const Modulus64 uint64 = 0xffffffff00000001

// » GENERATOR AND ORDER. Generator64 has multiplicative order 2^32, so
// » Generator64^(2^32) == 1 and no smaller power of two gives 1. poly.go's FFT
// » derives every root of unity it needs from this one element. Verify the order
// » in a test rather than trusting the constant — a wrong generator produces
// » proofs that verify locally and fail against the draft's test vectors, which
// » is the worst kind of bug to hunt.
const (
	Generator64      Field64 = 7277203076849721926
	GeneratorOrder64 uint64  = 1 << 32
	// EncodedSize64 is the number of bytes in the little-endian wire encoding.
	EncodedSize64 = 8
)

// Zero and One are the field's additive and multiplicative identities.
const (
	Zero64 Field64 = 0
	One64  Field64 = 1
)

// Add returns a + b mod p.
//
// TODO(week2): implement.
func Add(a, b Field64) Field64 {
	// EXERCISE-BEGIN
	// ─── EXERCISE 10: modular addition without a branch on overflow ──────────
	// Task: return (a + b) mod p using only uint64 arithmetic.
	//
	// The subtlety: a + b can exceed 2^64 and wrap. You must detect that wrap
	// and correct for it. Both operands are < p < 2^64, so the true sum is
	// < 2p < 2^65 and at most ONE subtraction of p is ever needed — but you have
	// to handle "wrapped" and "did not wrap but is ≥ p" as the same case.
	//
	// Write it first with an if, get the test green, then look at
	// math/bits.Add64 which gives you the carry directly. Then decide whether
	// you care: this function is called on the order of 10^9 times per
	// collection at fleet scale, and a mispredicted branch there is real money.
	// Measure before you believe it (go test -bench).
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0
}

// Sub returns a − b mod p.
//
// TODO(week2): implement — the mirror of Add; borrow instead of carry.
func Sub(a, b Field64) Field64 {
	return 0
}

// Neg returns −a mod p.
//
// TODO(week2): implement. Careful with a == 0: the answer is 0, not p.
func Neg(a Field64) Field64 {
	return 0
}

// Mul returns a · b mod p.
//
// TODO(week2): implement.
func Mul(a, b Field64) Field64 {
	// EXERCISE-BEGIN
	// ─── EXERCISE 11: multiplication, the interesting one ───────────────────
	// Task A (get it working): use math/bits.Mul64 for the 128-bit product, then
	//         reduce with math/big or a schoolbook 128-by-64 division. Slow and
	//         obviously correct. Get every test in this package green here.
	// Task B (get it fast): implement Goldilocks reduction using
	//             2^64  ≡ 2^32 − 1   (mod p)
	//             2^96  ≡ −1          (mod p)
	//         Write the product as lo + 2^64·hi, split hi into its high and low
	//         32-bit halves, and fold each piece down with shifts and adds. The
	//         standard derivation is three folds plus two conditional
	//         subtractions. Derive it on paper — this is the single most
	//         instructive half-hour in week 2, and the identity above is the
	//         whole trick.
	// Task C: benchmark A against B and put the numbers in docs/BENCHMARKS.md.
	//         Then compute how many field multiplications one Prio3Histogram
	//         report costs and multiply out: at 10^9 devices reporting hourly,
	//         what does the difference cost in CPU-hours per day? That
	//         calculation is the reason this field was chosen, and it is exactly
	//         the kind of reasoning the team does.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0
}

// Inv returns a^(p−2) mod p, the multiplicative inverse of a. Inv(0) is 0.
//
// TODO(week2): implement.
func Inv(a Field64) Field64 {
	// EXERCISE-BEGIN
	// ─── EXERCISE 12: inversion via Fermat's little theorem ─────────────────
	// For prime p, a^(p−1) ≡ 1, so a^(p−2) is the inverse.
	//
	// Task: square-and-multiply exponentiation using Mul. ~64 squarings and up
	//       to 64 multiplications, so roughly 100× the cost of a Mul — which is
	//       why every algorithm above this layer batches inversions or avoids
	//       them entirely. Note where inversion appears in poly.go (Lagrange
	//       interpolation, FFT scaling) and check you are not calling it in a
	//       per-report hot loop.
	// Do NOT use a data-dependent exponent loop that early-exits on zero bits
	//       and then claim constant time. p − 2 is a public constant here so
	//       timing does not leak anything secret, but get in the habit of
	//       knowing WHY a variable-time algorithm is safe rather than assuming.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0
}

// Exp returns a^e mod p.
//
// TODO(week2): implement — square-and-multiply; Inv is Exp(a, p−2).
func Exp(a Field64, e uint64) Field64 {
	return 0
}

// FromUint64 reduces v into the field.
//
// » Not the same as Field64(v): a uint64 can exceed p. Every place that turns
// » external data into a field element must go through here, or you will get
// » unreduced values that break the "always in [0, p)" invariant that Add and
// » Mul rely on. This is the most common source of "works on my machine,
// » fails on the test vectors".
func FromUint64(v uint64) Field64 {
	if v >= Modulus64 {
		return Field64(v % Modulus64)
	}
	return Field64(v)
}

// Encode appends a's little-endian 8-byte encoding to dst.
//
// » LITTLE-ENDIAN, and check this against the draft rather than guessing. VDAF
// » §6.1 specifies little-endian for field elements while the DAP messages in
// » internal/dap use big-endian network order for lengths and IDs. Mixing the
// » two is the classic reason test vectors fail with everything else correct,
// » and it costs an afternoon every time.
//
// TODO(week2): implement.
func Encode(dst []byte, a Field64) []byte {
	return dst
}

// Decode reads one field element from b, which must be exactly EncodedSize64
// bytes, and rejects encodings of values ≥ p.
//
// » REJECT, do not reduce. An attacker-supplied share encoding 2^64 − 1 must be
// » an error, not silently folded into the field: accepting non-canonical
// » encodings gives two distinct byte strings the same meaning, which breaks
// » the anti-replay report-ID deduplication in internal/dap/store.go. "Reject
// » non-canonical encodings" is a rule you should apply reflexively anywhere
// » bytes cross a trust boundary.
//
// TODO(week2): implement.
func Decode(b []byte) (Field64, error) {
	return 0, ErrTODO
}

// bigModulus is p as a big.Int, for the slow reference path in« EXERCISE 11»
// Task A and for cross-checking the fast path in tests.
//
// » Keeping a deliberately slow, obviously-correct reference implementation
// » around and testing the fast one against it is the standard way to build
// » confidence in optimised arithmetic. Do not delete it once Mul is fast —
// » make it the oracle in a fuzz test.
var bigModulus = new(big.Int).SetUint64(Modulus64)

// MulRef is the reference multiplication, via math/big. Correct and slow.
func MulRef(a, b Field64) Field64 {
	x := new(big.Int).SetUint64(uint64(a))
	y := new(big.Int).SetUint64(uint64(b))
	x.Mul(x, y)
	x.Mod(x, bigModulus)
	return Field64(x.Uint64())
}
