package dp

import (
	"crypto/rand"
	"encoding/binary"
	"math"
)

// randomFloat64 returns a uniform sample from [0, 1) using 53 random bits.
//
// » crypto/rand, not math/rand — and this is not paranoia. DP's guarantee is a
// » statement about a probability distribution; if an attacker can predict the
// » noise, ε is meaningless and you have published raw data with extra steps.
// » There is a whole genre of attack on DP implementations that seed a PRNG
// » predictably. Treat the noise source as key material.
func randomFloat64() float64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// » crypto/rand.Read on a healthy kernel does not fail. If it does, the
		// » only safe behaviour is to stop: silently degrading to a weak source
		// » would break the privacy guarantee invisibly.
		panic("dp: crypto/rand failed: " + err.Error())
	}
	// » Take the top 53 bits so every representable float64 in [0,1) is reachable
	// » with the correct probability. Dividing a full uint64 by 2^64 is the same
	// » idea but loses the low bits to rounding anyway.
	return float64(binary.BigEndian.Uint64(b[:])>>11) / (1 << 53)
}

// ─── continuous mechanisms ───────────────────────────────────────────────────

// Laplace is the ε-DP Laplace mechanism: add Lap(0, Δ₁/ε) to the aggregate,
// where Δ₁ is the L1 sensitivity of the query.
//
// » Use this when δ must be exactly 0 and you release a small, fixed number of
// » statistics. Its weakness is composition: k releases at ε each cost kε under
// » basic composition, and Laplace has no advanced-composition story as good as
// » Gaussian's. See accountant.go.
type Laplace struct{ P Params }

// Params implements Mechanism.
func (l Laplace) Params() Params { return l.P }

// LaplaceScale returns the scale parameter b of the Laplace distribution that
// makes a query with the given L1 sensitivity ε-differentially private.
//
// » Derive this once by hand rather than memorising it. The Laplace density is
// » f(x) ∝ exp(−|x|/b). For neighbouring datasets the true answers differ by at
// » most Δ₁, so the density ratio at any output is at most exp(Δ₁/b). Setting
// » that ≤ exp(ε) gives b ≥ Δ₁/ε. That derivation is a two-minute whiteboard
// » answer and it is asked.
//
// TODO(week1): implement.
func LaplaceScale(l1Sensitivity, epsilon float64) (float64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 1: the Laplace mechanism ──────────────────────────────────
	// Task A: return the scale b = Δ₁/ε, rejecting Δ₁ ≤ 0 and ε ≤ 0.
	// Task B: implement Laplace.Apply below by sampling Lap(0, b). The standard
	//         trick is inverse-CDF on a uniform u ∈ (0,1):
	//             sign · b · ln(1 − 2|u − ½|)
	//         Watch the boundary: u = 0 must not produce ±Inf. Reject and
	//         resample, or shift into the open interval.
	// Task C: write the test in noise_test.go that checks the empirical mean and
	//         variance of 10⁶ samples against 0 and 2b² within tolerance. A
	//         mechanism with the wrong variance is a silent privacy bug — it
	//         will not fail any type check and the aggregate will look fine.
	//
	// Careful, and worth an hour of your week-1 reading: sampling Laplace via
	// float64 arithmetic is NOT exactly the Laplace distribution. The gaps
	// between representable floats leak information about the noise, which
	// breaks ε — Mironov, "On Significance of the Least Significant Bits for
	// Differential Privacy" (CCS 2012). Production DP libraries either snap to
	// a lattice or use discrete mechanisms. You will implement the discrete
	// versions below; note in THREAT_MODEL.md that the float path is a
	// convenience for plotting, not the shipped path.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0, ErrTODO
}

// Apply implements Mechanism.
//
// TODO(week1): implement« — see EXERCISE 1 in LaplaceScale.»
func (l Laplace) Apply(aggregate, l1Sensitivity float64) (float64, error) {
	return 0, ErrTODO
}

// Gaussian is the (ε, δ)-DP Gaussian mechanism: add N(0, σ²) calibrated to the
// query's L2 sensitivity.
//
// » Why bother, when Laplace gives you δ = 0? Because of composition and
// » dimension. For a histogram with k buckets where one user touches one bucket,
// » Δ₁ = 1 but Δ₂ = 1 as well — no gain. But for vector-valued queries where a
// » user contributes to many coordinates (the gradient in DP-SGD, Project B),
// » Δ₁ grows like d while Δ₂ grows like √d, and Gaussian wins by a factor of √d.
// » Gaussian also has a clean Rényi-DP characterisation, which is what makes
// » thousands of composed releases accountable at all.
type Gaussian struct{ P Params }

// Params implements Mechanism.
func (g Gaussian) Params() Params { return g.P }

// GaussianSigma returns the standard deviation σ making a query with the given
// L2 sensitivity (ε, δ)-differentially private.
//
// TODO(week1): implement.
func GaussianSigma(l2Sensitivity, epsilon, delta float64) (float64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 2: the Gaussian mechanism, and its famous caveat ───────────
	// Task A: implement the classical bound σ ≥ Δ₂·√(2·ln(1.25/δ))/ε.
	//         Then read the fine print in Dwork & Roth Theorem A.1: it is only
	//         valid for ε ≤ 1. Reject ε > 1 with a clear error rather than
	//         silently returning a σ that does not provide the claimed
	//         guarantee. This exact bug ships in real code.
	// Task B: replace it with the ANALYTIC Gaussian mechanism (Balle & Wang,
	//         ICML 2018, arXiv 1805.06530), which is valid for all ε and gives
	//         a noticeably smaller σ. It needs a numeric search over Φ (the
	//         normal CDF) — bisection on their B⁺/B⁻ functions is enough.
	//         math.Erfc gives you Φ: Φ(x) = ½·erfc(−x/√2).
	// Task C: plot σ_classical / σ_analytic against ε ∈ [0.1, 10] and put the
	//         plot in docs/BENCHMARKS.md. "How much accuracy does the tighter
	//         analysis buy?" is exactly the kind of question this team lives on.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0, ErrTODO
}

// Apply implements Mechanism.
//
// TODO(week1): implement« — see EXERCISE 2 in GaussianSigma.»
func (g Gaussian) Apply(aggregate, l2Sensitivity float64) (float64, error) {
	return 0, ErrTODO
}

// standardNormal returns one sample from N(0, 1).
//
// » Box–Muller is fine here and is three lines. Do not reach for a library.
func standardNormal() float64 {
	// » u1 must be strictly positive or Log blows up; resample the measure-zero
	// » case rather than nudging it, so the distribution stays exact.
	u1 := randomFloat64()
	for u1 == 0 {
		u1 = randomFloat64()
	}
	u2 := randomFloat64()
	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

// ─── discrete mechanisms: what the private path actually uses ────────────────

// IntMechanism perturbs an integer aggregate. The private path aggregates VDAF
// shares in a prime field, so noise must be integer-valued to be added inside
// the field without ever materialising a real number.
//
// » This is the reason the discrete mechanisms are not an academic curiosity
// » here. Prio3 sums elements of GF(p); "add Gaussian noise" is not an
// » operation that field has. The discrete Gaussian is the right primitive, it
// » has essentially the same utility as its continuous cousin, and it sidesteps
// » the floating-point leakage from« EXERCISE 1» entirely. Google's and Apple's
// » shipped aggregation stacks both use discrete noise for exactly this reason.
type IntMechanism interface {
	ApplyInt(aggregate, sensitivity int64) (int64, error)
	Params() Params
}

// DiscreteGaussian is the discrete Gaussian mechanism of Canonne, Kamath &
// Steinke (arXiv 2004.00010): Pr[X = k] ∝ exp(−k²/(2σ²)) over the integers.
type DiscreteGaussian struct{ P Params }

// Params implements IntMechanism.
func (d DiscreteGaussian) Params() Params { return d.P }

// ApplyInt implements IntMechanism.
//
// TODO(week1): implement« — see EXERCISE 3 in SampleDiscreteGaussian.»
func (d DiscreteGaussian) ApplyInt(aggregate, sensitivity int64) (int64, error) {
	return 0, ErrTODO
}

// SampleDiscreteGaussian draws from the discrete Gaussian with the given σ.
//
// TODO(week1): implement.
func SampleDiscreteGaussian(sigma float64) (int64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 3: the discrete Gaussian (the one that ships) ──────────────
	// This is the most valuable single exercise in week 1. It is a page of code
	// and it is the mechanism the private path needs.
	//
	// Read Canonne–Kamath–Steinke §4–5 and build up in this order, each with a
	// test, because a bug in the bottom layer is invisible from the top:
	//
	//   1. bernoulliExp(-x) for x ∈ [0,1]: their Algorithm 1. Uses only
	//      integer/rational arithmetic and unbiased coin flips.
	//   2. bernoulliExp(-x) for arbitrary x ≥ 0: split into ⌊x⌋ + frac.
	//   3. SampleDiscreteLaplace(t/s) by rejection sampling on a geometric.
	//   4. SampleDiscreteGaussian(σ): propose from a discrete Laplace with
	//      scale ⌊σ⌋+1, accept with probability
	//          exp(−(|Y| − σ²/t)² / (2σ²))
	//      using the Bernoulli routine from step 2.
	//
	// Use math/big.Rat or integer pairs, NOT float64, inside the samplers —
	// exact rational arithmetic is the entire point of choosing this mechanism.
	// (crypto/rand + math/big is all you need; no new module dependency.)
	//
	// Verify with a χ² goodness-of-fit test against the exact pmf for σ = 1, 3,
	// 10 over a truncated support. Do not settle for "the mean looks like zero".
	//
	// Then answer for docs/PRIVACY.md: your field is GF(2^64 − 2^32 + 1) and
	// noise can be negative. How do you represent −5 in the field, and what
	// happens to the aggregate if the true sum plus noise wraps past p?
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0, ErrTODO
}

// SampleDiscreteLaplace draws from the discrete Laplace (two-sided geometric)
// with scale t/s, i.e. Pr[X = k] ∝ exp(−|k|·s/t).
//
// TODO(week1): implement« — see EXERCISE 3».
func SampleDiscreteLaplace(t, s uint64) (int64, error) {
	return 0, ErrTODO
}

// bernoulliExp returns a coin flip that is true with probability exp(−num/den).
//
// TODO(week1): implement the Bernoulli routine«, see EXERCISE 3 step 1».
func bernoulliExp(num, den uint64) (bool, error) {
	return false, ErrTODO
}
