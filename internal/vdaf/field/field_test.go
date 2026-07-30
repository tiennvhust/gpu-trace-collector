package field

import (
	"math/rand"
	"testing"
)

// » ORDER OF ATTACK for week 2. Run `go test ./internal/vdaf/field` and make
// » these pass top to bottom. Each one is a few lines of implementation and
// » nothing above this package can work until they are all green.

func TestAddIdentityAndCommutativity(t *testing.T) {
	if got := Add(One64, Zero64); got != One64 {
		t.Errorf("1 + 0 = %d, want 1", got)
	}
	// The wrap case: (p−1) + 1 must be 0, not 2^64 mod 2^64.
	pMinus1 := Field64(Modulus64 - 1)
	if got := Add(pMinus1, One64); got != Zero64 {
		t.Errorf("(p−1) + 1 = %d, want 0", got)
	}
	if got := Add(pMinus1, pMinus1); got != Field64(Modulus64-2) {
		t.Errorf("(p−1) + (p−1) = %d, want p−2", got)
	}
}

func TestSubAndNeg(t *testing.T) {
	if got := Sub(Zero64, One64); got != Field64(Modulus64-1) {
		t.Errorf("0 − 1 = %d, want p−1", got)
	}
	if got := Neg(Zero64); got != Zero64 {
		t.Errorf("−0 = %d, want 0 (not p)", got)
	}
	if got := Add(Field64(12345), Neg(Field64(12345))); got != Zero64 {
		t.Errorf("a + (−a) = %d, want 0", got)
	}
}

func TestMulAgainstReference(t *testing.T) {
	// » MulRef is the math/big oracle. This test is the whole reason to keep a
	// » slow reference implementation: once it passes on random input, the fast
	// » Goldilocks reduction from« EXERCISE 11 Task B» is trustworthy.
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 20_000; i++ {
		a := FromUint64(r.Uint64())
		b := FromUint64(r.Uint64())
		if got, want := Mul(a, b), MulRef(a, b); got != want {
			t.Fatalf("Mul(%d, %d) = %d, want %d", a, b, got, want)
		}
	}
}

func TestMulEdgeCases(t *testing.T) {
	pMinus1 := Field64(Modulus64 - 1)
	cases := []struct{ a, b Field64 }{
		{0, 0},
		{0, pMinus1},
		{1, pMinus1},
		// largest × largest: the reduction's worst case
		{pMinus1, pMinus1},
		// exercises the 2^64 ≡ 2^32 − 1 fold directly
		{1 << 32, 1 << 32},
		// lands just under p, then wraps by exactly 1
		{Field64(Modulus64 / 2), 2},
		{Field64(Modulus64/2 + 1), 2},
	}
	for _, tc := range cases {
		if got, want := Mul(tc.a, tc.b), MulRef(tc.a, tc.b); got != want {
			t.Errorf("Mul(%d, %d) = %d, want %d", tc.a, tc.b, got, want)
		}
	}
}

func TestInvIsAMultiplicativeInverse(t *testing.T) {
	if got := Inv(Zero64); got != Zero64 {
		t.Errorf("Inv(0) = %d, want 0 by convention", got)
	}
	r := rand.New(rand.NewSource(2))
	for i := 0; i < 500; i++ {
		a := FromUint64(r.Uint64())
		if a == 0 {
			continue
		}
		if got := Mul(a, Inv(a)); got != One64 {
			t.Fatalf("a · a⁻¹ = %d for a = %d, want 1", got, a)
		}
	}
}

func TestGeneratorHasOrder2To32(t *testing.T) {
	// » Verify the constant instead of trusting it. g^(2^32) must be 1, and
	// » g^(2^31) must NOT be — otherwise the order is smaller than claimed and
	// » every root of unity derived from it is wrong.
	if got := Exp(Generator64, GeneratorOrder64); got != One64 {
		t.Errorf("g^(2^32) = %d, want 1", got)
	}
	if got := Exp(Generator64, GeneratorOrder64/2); got == One64 {
		t.Error("g^(2^31) = 1, so the generator's order is not 2^32")
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for i := 0; i < 1000; i++ {
		a := FromUint64(r.Uint64())
		enc := Encode(nil, a)
		if len(enc) != EncodedSize64 {
			t.Fatalf("Encode produced %d bytes, want %d", len(enc), EncodedSize64)
		}
		got, err := Decode(enc)
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if got != a {
			t.Fatalf("round trip: got %d, want %d", got, a)
		}
	}
}

func TestEncodeIsLittleEndian(t *testing.T) {
	// » Pin the byte order with a known answer. VDAF §6.1 says little-endian for
	// » field elements; DAP uses big-endian for message lengths. One known-answer
	// » test in each package saves you the afternoon described in field.go.
	enc := Encode(nil, Field64(1))
	want := []byte{1, 0, 0, 0, 0, 0, 0, 0}
	for i := range want {
		if i >= len(enc) || enc[i] != want[i] {
			t.Fatalf("Encode(1) = %v, want %v (little-endian)", enc, want)
		}
	}
}

func TestDecodeRejectsNonCanonicalEncodings(t *testing.T) {
	// p itself, and 2^64 − 1: both ≥ p and both must be rejected, not reduced.
	for _, b := range [][]byte{
		{0x01, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff}, // == p, little-endian
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // 2^64 − 1
	} {
		if v, err := Decode(b); err == nil {
			t.Errorf("Decode(%x) = %d, want an error (non-canonical)", b, v)
		}
	}
	if _, err := Decode([]byte{1, 2, 3}); err == nil {
		t.Error("Decode of a short slice should fail")
	}
}

func TestPolyEvalKnownAnswer(t *testing.T) {
	// p(x) = 3 + 2x + x², p(2) = 3 + 4 + 4 = 11
	p := Poly{3, 2, 1}
	if got := p.Eval(2); got != 11 {
		t.Errorf("p(2) = %d, want 11", got)
	}
	if got := (Poly{}).Eval(5); got != Zero64 {
		t.Errorf("zero polynomial evaluated to %d, want 0", got)
	}
}

func TestInterpolateRoundTrip(t *testing.T) {
	xs := []Field64{1, 2, 3, 4, 5}
	ys := []Field64{7, 13, 999, 4, 2024}
	p, err := Interpolate(xs, ys)
	if err != nil {
		t.Fatalf("Interpolate: %v", err)
	}
	if d := p.Degree(); d >= len(xs) {
		t.Errorf("interpolated degree %d, want < %d", d, len(xs))
	}
	for i := range xs {
		if got := p.Eval(xs[i]); got != ys[i] {
			t.Errorf("p(%d) = %d, want %d", xs[i], got, ys[i])
		}
	}
}

func TestInterpolateRejectsDuplicateX(t *testing.T) {
	if _, err := Interpolate([]Field64{1, 1}, []Field64{2, 3}); err == nil {
		t.Error("duplicate x values should be rejected, not silently mishandled")
	}
}

func BenchmarkMul(b *testing.B) {
	// » Run `go test -bench=Mul ./internal/vdaf/field` before and after
	// »« EXERCISE 11 Task B» and paste both numbers into docs/BENCHMARKS.md.
	x, y := Field64(0x1234_5678_9abc_def0), Field64(0x0fed_cba9_8765_4321)
	var acc Field64
	for i := 0; i < b.N; i++ {
		acc = Mul(x, y)
	}
	_ = acc
}

func BenchmarkMulRef(b *testing.B) {
	x, y := Field64(0x1234_5678_9abc_def0), Field64(0x0fed_cba9_8765_4321)
	var acc Field64
	for i := 0; i < b.N; i++ {
		acc = MulRef(x, y)
	}
	_ = acc
}
