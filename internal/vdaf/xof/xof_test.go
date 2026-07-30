package xof

import (
	"bytes"
	"testing"

	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/field"
)

// » These tests are written against the XOF INTERFACE, so they run against both
// » implementations. That is the point: HMACCounter passes them today, and when
// » TurboShake128 lands it must pass the same set. Only the test-vector test at
// » the bottom distinguishes the two.

func implementations() map[string]func() XOF {
	return map[string]func() XOF{
		"hmac-counter":   func() XOF { return &HMACCounter{} },
		"turboshake-128": func() XOF { return &TurboShake128{} },
	}
}

func TestDeterministic(t *testing.T) {
	for name, mk := range implementations() {
		t.Run(name, func(t *testing.T) {
			seed := Seed{1, 2, 3}
			a, err := Derive(mk(), seed, []byte("dst"), []byte("binder"), 100)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			b, err := Derive(mk(), seed, []byte("dst"), []byte("binder"), 100)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			if !bytes.Equal(a, b) {
				t.Error("same inputs produced different output: the client and the helper would derive different shares")
			}
		})
	}
}

func TestDomainSeparationChangesOutput(t *testing.T) {
	// » The security-relevant test in this file. If two different dst values
	// » produce the same stream, Prio3's soundness argument does not hold — the
	// » query randomness and the share randomness would be the same bytes, and a
	// » malicious client could exploit the coincidence.
	for name, mk := range implementations() {
		t.Run(name, func(t *testing.T) {
			seed := Seed{9}
			a, err := Derive(mk(), seed, []byte("dst-a"), []byte("bind"), 64)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			b, err := Derive(mk(), seed, []byte("dst-b"), []byte("bind"), 64)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			if bytes.Equal(a, b) {
				t.Error("different dst produced identical output")
			}

			c, err := Derive(mk(), seed, []byte("dst-a"), []byte("other"), 64)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			if bytes.Equal(a, c) {
				t.Error("different binder produced identical output")
			}
		})
	}
}

func TestNoLengthExtensionAmbiguity(t *testing.T) {
	// » (dst="ab", binder="c") and (dst="a", binder="bc") concatenate to the same
	// » bytes. Without length prefixing they key the XOF identically. This is a
	// » canonicalisation bug of exactly the kind that gets CVEs, and it costs one
	// » length prefix to prevent.
	for name, mk := range implementations() {
		t.Run(name, func(t *testing.T) {
			seed := Seed{7}
			a, err := Derive(mk(), seed, []byte("ab"), []byte("c"), 32)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			b, err := Derive(mk(), seed, []byte("a"), []byte("bc"), 32)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			if bytes.Equal(a, b) {
				t.Error("ambiguous encoding: length-prefix dst and binder")
			}
		})
	}
}

func TestReadIsAStream(t *testing.T) {
	// One 64-byte read must equal four 16-byte reads: callers pull field
	// elements one at a time and must not get a fresh block each call.
	for name, mk := range implementations() {
		t.Run(name, func(t *testing.T) {
			seed := Seed{4}

			whole := make([]byte, 64)
			x := mk()
			if err := x.Init(seed, []byte("d"), nil); err != nil {
				t.Fatalf("Init: %v", err)
			}
			if _, err := x.Read(whole); err != nil {
				t.Fatalf("Read: %v", err)
			}

			piecemeal := make([]byte, 0, 64)
			y := mk()
			if err := y.Init(seed, []byte("d"), nil); err != nil {
				t.Fatalf("Init: %v", err)
			}
			for i := 0; i < 4; i++ {
				chunk := make([]byte, 16)
				if _, err := y.Read(chunk); err != nil {
					t.Fatalf("Read: %v", err)
				}
				piecemeal = append(piecemeal, chunk...)
			}

			if !bytes.Equal(whole, piecemeal) {
				t.Error("Read is not a continuous stream")
			}
		})
	}
}

func TestFieldElementsAreCanonical(t *testing.T) {
	// » Every sampled element must be < p. This catches the "% p" shortcut only
	// » if the modulus is not applied — the real defence against bias is
	// »« EXERCISE 16»'s rejection sampling, which this test cannot prove. Note that
	// » honestly: a statistical test for a 2^-32 bias would need ~2^64 samples,
	// » so correctness here comes from reading the code, not from the test. That
	// » asymmetry is worth internalising about crypto code generally.
	for name, mk := range implementations() {
		t.Run(name, func(t *testing.T) {
			x := mk()
			if err := x.Init(Seed{2}, []byte("d"), nil); err != nil {
				t.Fatalf("Init: %v", err)
			}
			out := make([]field.Field64, 512)
			if err := x.FillField64(out); err != nil {
				t.Fatalf("FillField64: %v", err)
			}
			for i, v := range out {
				if uint64(v) >= field.Modulus64 {
					t.Fatalf("out[%d] = %d is not reduced", i, v)
				}
			}
		})
	}
}

func TestTurboShake128MatchesDraftKAT(t *testing.T) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 17: pin the XOF with a known-answer test ──────────────────
	// Before wiring TurboShake128 into Prio3, pin it here.
	//
	// Task A: copy one TurboSHAKE128 KAT from the KangarooTwelve draft
	//         (draft-irtf-cfrg-kangarootwelve, Appendix A) and assert on it.
	// Task B: then copy the XofTurboShake128 vector from the VDAF draft's own
	//         test vectors and assert on that — it exercises the seed/dst/binder
	//         framing on top of the raw permutation, which is the part you are
	//         most likely to get wrong.
	// Remove the Skip when you add the first vector.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	t.Skip("EXERCISE 17: add a TurboSHAKE128 known-answer vector")
}
