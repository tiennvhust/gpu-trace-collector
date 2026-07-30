package dp

import (
	"math"
	"testing"
)

func TestCountSensitivity(t *testing.T) {
	l1, l2 := Count{}.Sensitivity()
	if l1 != 1 || l2 != 1 {
		t.Fatalf("Count sensitivity = (%v, %v), want (1, 1)", l1, l2)
	}
}

func TestBoundedSumSensitivity(t *testing.T) {
	// » Fill in the `want` column yourself as part of« EXERCISE 4», choosing
	// » unbounded-DP (add/remove a row) or bounded-DP (replace a row) — and then
	// » make every case consistent with that choice. The point of the table is
	// » to force you to state the definition; two of these rows are only
	// » unambiguous once you have.
	cases := []struct {
		name   string
		q      BoundedSum
		wantL1 float64
		wantL2 float64
	}{
		{"unit interval, one row", BoundedSum{Lo: 0, Hi: 1, MaxContributions: 1}, 1, 1},
		{"utilisation percent", BoundedSum{Lo: 0, Hi: 100, MaxContributions: 1}, 100, 100},
		{"symmetric around zero", BoundedSum{Lo: -50, Hi: 50, MaxContributions: 1}, 50, 50},
		{"asymmetric, negative floor", BoundedSum{Lo: -10, Hi: 3, MaxContributions: 1}, 10, 10},
		{"three rows per device", BoundedSum{Lo: 0, Hi: 1, MaxContributions: 3}, 3, math.Sqrt(3)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l1, l2 := tc.q.Sensitivity()
			if math.Abs(l1-tc.wantL1) > 1e-9 {
				t.Errorf("L1 = %v, want %v", l1, tc.wantL1)
			}
			if math.Abs(l2-tc.wantL2) > 1e-9 {
				t.Errorf("L2 = %v, want %v", l2, tc.wantL2)
			}
		})
	}
}

func TestBoundedSumClamp(t *testing.T) {
	q := BoundedSum{Lo: -5, Hi: 10, MaxContributions: 1}
	for _, tc := range []struct{ in, want float64 }{
		{0, 0}, {-5, -5}, {10, 10}, {-1000, -5}, {1e9, 10},
	} {
		if got := q.Clamp(tc.in); got != tc.want {
			t.Errorf("Clamp(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestHistogramSensitivity(t *testing.T) {
	// One device increments one bucket by one: L1 = L2 = 1 regardless of how
	// many buckets exist.
	l1, l2 := Histogram{Buckets: 1024, MaxBuckets: 1}.Sensitivity()
	if l1 != 1 || l2 != 1 {
		t.Errorf("single-bucket histogram sensitivity = (%v, %v), want (1, 1)", l1, l2)
	}

	// » And the punchline that surprises people the first time: bucket COUNT
	// » does not enter sensitivity at all, so per-bucket noise is independent of
	// » how finely you slice. What bucket count costs you is that each bucket's
	// » true value is smaller, so the same absolute noise is a larger relative
	// » error. That distinction — absolute error constant, relative error
	// » exploding — is the right way to explain "why can't we just add more
	// » buckets" to a product manager.
	l1, l2 = Histogram{Buckets: 8, MaxBuckets: 4}.Sensitivity()
	if l1 != 4 {
		t.Errorf("L1 with MaxBuckets=4: got %v, want 4", l1)
	}
	if math.Abs(l2-2) > 1e-9 {
		t.Errorf("L2 with MaxBuckets=4: got %v, want 2 (=√4)", l2)
	}
}

func TestHistogramPerBucketStdDev(t *testing.T) {
	// At ε = 1, Δ₁ = 1, Laplace scale b = 1 and StdDev = √2.
	h := Histogram{Buckets: 64, MaxBuckets: 1}
	got, err := h.PerBucketStdDev(Laplace{P: Params{Epsilon: 1}})
	if err != nil {
		t.Fatalf("PerBucketStdDev: %v", err)
	}
	if math.Abs(got-math.Sqrt2) > 1e-9 {
		t.Errorf("per-bucket stddev = %v, want %v", got, math.Sqrt2)
	}
}
