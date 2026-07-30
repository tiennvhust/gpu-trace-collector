package dp

import (
	"math"
	"testing"
)

// » HOW TO TEST A RANDOM ALGORITHM. You cannot assert on a sample. You assert
// » on the DISTRIBUTION, which means: many samples, a statistic with a known
// » expected value, and a tolerance you derived rather than tuned until green.
// » For a mean over n samples of a distribution with variance v, the standard
// » error is √(v/n) — so a 4-sigma tolerance is 4√(v/n). Write that formula in
// » the test instead of a magic 0.05, and the test stays correct when you change
// » n. Tests whose tolerance was tuned until they passed are the main reason
// » subtly-wrong DP implementations survive code review.

const samples = 200_000

func TestLaplaceScale(t *testing.T) {
	// b = Δ₁/ε
	got, err := LaplaceScale(2, 0.5)
	if err != nil {
		t.Fatalf("LaplaceScale: %v", err)
	}
	if math.Abs(got-4) > 1e-12 {
		t.Fatalf("LaplaceScale(2, 0.5) = %v, want 4", got)
	}
}

func TestLaplaceScaleRejectsBadInput(t *testing.T) {
	for _, tc := range []struct{ sens, eps float64 }{
		{0, 1}, {-1, 1}, {1, 0}, {1, -1}, {math.NaN(), 1},
	} {
		if _, err := LaplaceScale(tc.sens, tc.eps); err == nil {
			t.Errorf("LaplaceScale(%v, %v): want error, got nil", tc.sens, tc.eps)
		}
	}
}

func TestLaplaceMomentsMatchTheDistribution(t *testing.T) {
	const (
		sensitivity = 1.0
		epsilon     = 1.0
	)
	b, err := LaplaceScale(sensitivity, epsilon)
	if err != nil {
		t.Fatalf("LaplaceScale: %v", err)
	}
	mech := Laplace{P: Params{Epsilon: epsilon}}

	sum, sumSq := 0.0, 0.0
	for i := 0; i < samples; i++ {
		v, err := mech.Apply(0, sensitivity)
		if err != nil {
			t.Fatalf("Apply: %v", err)
		}
		sum += v
		sumSq += v * v
	}
	mean := sum / samples
	variance := sumSq/samples - mean*mean

	// Var(Lap(0,b)) = 2b². Fourth moment gives Var of the sample variance;
	// 4 standard errors is a ~1-in-15000 flake per assertion.
	wantVar := 2 * b * b
	if tol := 4 * math.Sqrt(wantVar/samples); math.Abs(mean) > tol {
		t.Errorf("mean = %v, want 0 ± %v", mean, tol)
	}
	if tol := 0.05 * wantVar; math.Abs(variance-wantVar) > tol {
		t.Errorf("variance = %v, want %v ± %v", variance, wantVar, tol)
	}
}

func TestGaussianSigmaClassicalBound(t *testing.T) {
	// σ = Δ₂·√(2·ln(1.25/δ))/ε
	const eps, delta = 0.5, 1e-6
	want := math.Sqrt(2*math.Log(1.25/delta)) / eps
	got, err := GaussianSigma(1, eps, delta)
	if err != nil {
		t.Fatalf("GaussianSigma: %v", err)
	}
	// » Once you switch to the analytic Gaussian mechanism (EXERCISE 2 Task B)
	// » this assertion should be relaxed to `got <= want`: the analytic σ is
	// » strictly smaller, and that is the improvement, not a regression.
	if math.Abs(got-want)/want > 0.01 {
		t.Fatalf("GaussianSigma(1, %v, %v) = %v, want ≈ %v", eps, delta, got, want)
	}
}

func TestGaussianSigmaRejectsEpsilonAboveOne(t *testing.T) {
	// » The classical analysis is only valid for ε ≤ 1. Returning a number
	// » anyway is the bug this test exists to prevent. Delete this test only
	// » when you have implemented the analytic mechanism, which is valid for all
	// » ε — and then replace it with a test that ε = 4 gives a sane σ.
	if _, err := GaussianSigma(1, 4, 1e-6); err == nil {
		t.Fatal("GaussianSigma with eps=4: want error from the classical bound, got nil")
	}
}

func TestDiscreteGaussianMatchesExactPMF(t *testing.T) {
	const sigma = 3.0

	counts := map[int64]int{}
	for i := 0; i < samples; i++ {
		v, err := SampleDiscreteGaussian(sigma)
		if err != nil {
			t.Fatalf("SampleDiscreteGaussian: %v", err)
		}
		counts[v]++
	}

	// Exact pmf: Pr[X=k] ∝ exp(−k²/(2σ²)), normalised over |k| ≤ 6σ (the tail
	// beyond that is below 1e-9 and contributes nothing at this sample size).
	const bound = 18
	norm := 0.0
	for k := int64(-bound); k <= bound; k++ {
		norm += math.Exp(-float64(k*k) / (2 * sigma * sigma))
	}

	// Pearson χ² over the buckets with adequate expected counts.
	chi2, df := 0.0, 0
	for k := int64(-bound); k <= bound; k++ {
		expected := samples * math.Exp(-float64(k*k)/(2*sigma*sigma)) / norm
		if expected < 25 {
			continue
		}
		d := float64(counts[k]) - expected
		chi2 += d * d / expected
		df++
	}
	// » Critical value for the χ² distribution at the 0.999 level. Look it up
	// » for your actual df rather than copying a number: df here is about 25, and
	// » χ²(25, 0.999) ≈ 52.6. If you widen `bound` you must widen this too.
	if chi2 > 52.6 {
		t.Errorf("χ² = %v over %d buckets: sampled distribution does not match the discrete Gaussian pmf", chi2, df)
	}
}

func TestDiscreteGaussianIsSymmetric(t *testing.T) {
	// » A cheap, fast test that catches the most common bug in a rejection
	// » sampler: getting the sign of the proposal wrong so the distribution is
	// » one-sided. Keep fast structural tests like this alongside the slow
	// » distributional ones — they are what you actually run while iterating.
	pos, neg := 0, 0
	for i := 0; i < 20_000; i++ {
		v, err := SampleDiscreteGaussian(5)
		if err != nil {
			t.Fatalf("SampleDiscreteGaussian: %v", err)
		}
		switch {
		case v > 0:
			pos++
		case v < 0:
			neg++
		}
	}
	if diff := math.Abs(float64(pos - neg)); diff > 4*math.Sqrt(float64(pos+neg)) {
		t.Errorf("asymmetric: %d positive vs %d negative", pos, neg)
	}
}

func TestBernoulliExpProbability(t *testing.T) {
	// Pr[true] should be exp(−1/2) ≈ 0.6065.
	const n = 100_000
	want := math.Exp(-0.5)
	hits := 0
	for i := 0; i < n; i++ {
		ok, err := bernoulliExp(1, 2)
		if err != nil {
			t.Fatalf("bernoulliExp: %v", err)
		}
		if ok {
			hits++
		}
	}
	got := float64(hits) / n
	if tol := 4 * math.Sqrt(want*(1-want)/n); math.Abs(got-want) > tol {
		t.Errorf("bernoulliExp(1,2) = %v, want %v ± %v", got, want, tol)
	}
}

func TestRandomFloat64InRange(t *testing.T) {
	// » This one passes out of the box. It is here because "the uniform sampler
	// » is correct" is the assumption every other test in this file rests on.
	for i := 0; i < 10_000; i++ {
		v := randomFloat64()
		if v < 0 || v >= 1 {
			t.Fatalf("randomFloat64 = %v, want [0, 1)", v)
		}
	}
}
