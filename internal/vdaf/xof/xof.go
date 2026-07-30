// Package xof provides the extendable-output functions VDAF uses to expand
// short seeds into long pseudorandom strings.
//
// » WHY THIS PACKAGE EXISTS, AND IT IS NOT AN IMPLEMENTATION DETAIL. A
// » Prio3Histogram report over 1024 buckets contains an input vector of 1024
// » field elements — 8 KiB — plus a proof. With two aggregators, naive additive
// » sharing means uploading 8 KiB to EACH of them: 16 KiB per report.
// »
// » The fix: the helper's share is not transmitted at all. The client sends the
// » helper a 16-byte SEED, the helper expands it with this XOF into the same
// » 8 KiB of pseudorandom field elements the client used, and the client sends
// » the leader only the difference (true value − expanded share). Report size
// » collapses from 16 KiB to 8 KiB + 16 bytes.
// »
// » On a device that wakes up, reports, and goes back to sleep on a battery,
// » bytes-on-wire IS the dominant cost — the radio dwarfs the CPU. So this seed
// » trick is the difference between a protocol that ships to a billion devices
// » and one that does not, and "quantify the privacy tax" in docs/BENCHMARKS.md
// » is mostly a measurement of this. Make sure your benchmark reports report
// » size WITH and WITHOUT the seed optimisation.
//
// » Spec: draft-irtf-cfrg-vdaf §6.2 (XOFs) and §7.2 (how Prio3 uses them).
package xof

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/field"
)

// ErrTODO marks an unimplemented scaffold function.
var ErrTODO = errors.New("xof: not implemented«, see the EXERCISE block above this function»")

// SeedSize is the length of a VDAF seed in bytes.
//
// » 16 bytes = 128 bits of security, and it is 16 rather than 32 because this
// » value is multiplied by the number of reports on the wire. Every constant in
// » a device-to-cloud protocol is a bandwidth decision.
const SeedSize = 16

// Seed is a VDAF seed.
type Seed [SeedSize]byte

// XOF is an extendable-output function keyed by a seed and domain-separated by
// a distinguishing string (dst) plus a binder.
//
// » THE THREE INPUTS EXIST FOR THREE DIFFERENT REASONS, and conflating them is a
// » real vulnerability rather than a style problem:
// »   seed    — the secret. Compromise it and the share it expands is recovered.
// »   dst     — domain separation. The same seed is expanded for several
// »             purposes (share expansion, the joint randomness, the query
// »             randomness). Without distinct dst values, two of those outputs
// »             would be identical and the proof system's soundness argument
// »             collapses, because a cheater could make one cancel the other.
// »   binder  — binds the output to a context: the task ID, the report nonce.
// »             This is what stops a report generated for task A from being
// »             replayed into task B.
// » If you ever find yourself passing an empty dst "because it works", stop.
type XOF interface {
	// Init prepares the XOF; it may be called only once per XOF value.
	Init(seed Seed, dst, binder []byte) error

	// Read fills b with the next bytes of output.
	Read(b []byte) (int, error)

	// NextField64 returns the next field element from the output stream.
	NextField64() (field.Field64, error)

	// FillField64 fills out with successive field elements.
	FillField64(out []field.Field64) error
}

// ─── the conformant XOF ──────────────────────────────────────────────────────

// TurboShake128 is the XOF the VDAF draft specifies (XofTurboShake128).
//
// » THE TRAP, and it is worth reading before you spend a day on it: Go's
// » standard library has SHAKE128 (crypto/sha3 as of Go 1.24) but NOT
// » TurboSHAKE128. They are not the same function. TurboSHAKE uses 12 rounds of
// » the Keccak-f[1600] permutation instead of 24, and a different domain
// » separation byte. Substituting SHAKE128 gives you a perfectly good XOF that
// » produces completely different bytes — so everything will work end to end,
// » your own tests will pass, and the draft's Appendix C test vectors will fail
// » with no useful diagnostic.
// »
// » Passing those vectors is the highest-signal artifact in the whole 8-week
// » plan (it is objective proof you read a spec correctly), so this is where the
// » effort goes.
type TurboShake128 struct {
	// » Hold whatever state your chosen implementation needs. If you write the
	// » permutation yourself, that is a [25]uint64 lane array plus a rate
	// » offset; if you wrap a library, it is that library's handle.
	initialised bool
}

// Init implements XOF.
//
// TODO(week2): implement.
func (x *TurboShake128) Init(seed Seed, dst, binder []byte) error {
	// EXERCISE-BEGIN
	// ─── EXERCISE 15: XofTurboShake128 (the test-vector gate) ────────────────
	// Read VDAF draft §6.2.1 for the exact construction. It is roughly:
	//     TurboSHAKE128(D=1, M = [len(dst)] ‖ dst ‖ seed ‖ binder)
	// — but take the byte layout from the draft, not from this comment, because
	// getting it from a summary is exactly how people fail the vectors.
	//
	// Two routes, pick deliberately:
	//
	// ROUTE A — write TurboSHAKE128 yourself, ~150 lines, no dependencies.
	//   You need Keccak-f[1600] with the round count parameterised, then sponge
	//   absorb/squeeze. Go's crypto/sha3 internals are a good reference for the
	//   permutation. Slower to get right, but you end up genuinely understanding
	//   a sponge construction, and it keeps this project dependency-free — which
	//   is itself a nice line in the README.
	//   Spec: https://datatracker.ietf.org/doc/draft-irtf-cfrg-kangarootwelve/
	//
	// ROUTE B — take the dependency: github.com/cloudflare/circl/xof/k12 or
	//   filippo.io/keccak. Ten minutes, and it lets you spend week 2 on Prio3
	//   itself, which is the higher-value material. Defensible choice; say so in
	//   the README rather than leaving it unexplained.
	//
	// Recommendation: Route B first so the vectors go green and you learn the
	// protocol, then Route A later if you want it. Do not let the sponge block
	// week 2.
	//
	// Verify against the TurboSHAKE128 KATs in the KangarooTwelve draft BEFORE
	// wiring it into Prio3. Debugging a sponge through three layers of secret
	// sharing is misery; debugging it against a KAT is five minutes.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return ErrTODO
}

// Read implements XOF.
//
// TODO(week2): implement« — see EXERCISE 15».
func (x *TurboShake128) Read(b []byte) (int, error) { return 0, ErrTODO }

// NextField64 implements XOF.
//
// TODO(week2): implement rejection sampling over the byte stream«, see EXERCISE 16».
func (x *TurboShake128) NextField64() (field.Field64, error) { return 0, ErrTODO }

// FillField64 implements XOF.
//
// TODO(week2): implement.
func (x *TurboShake128) FillField64(out []field.Field64) error { return ErrTODO }

// ─── a stdlib-only stand-in, so the plumbing above can be built first ────────

// HMACCounter is an XOF built from HMAC-SHA256 in counter mode. It is a
// perfectly sound PRF and it is NOT draft-conformant.
//
// » Deliberately here so you can build and test all of Prio3 — sharing, the
// » FLP, the aggregators, the DAP round trip — with a working XOF before
// » TurboSHAKE exists. Your own end-to-end tests will pass with this; only the
// » draft's test vectors will not.
// »
// » Keep it after TurboShake128 works, and keep testing against BOTH. If a
// » Prio3 test passes with one XOF and fails with the other, the bug is in your
// » XOF usage (wrong dst, wrong binder, output consumed in the wrong order) and
// » not in the XOF — which is a diagnostic you will be glad of.
type HMACCounter struct {
	key     []byte
	counter uint64
	buf     []byte
	off     int
}

// Init implements XOF.
func (x *HMACCounter) Init(seed Seed, dst, binder []byte) error {
	if len(dst) == 0 {
		// » Fail closed on a missing domain separator rather than accepting it.
		// » See the note on the XOF interface: an empty dst is a soundness bug,
		// » so the type refuses to represent one.
		return errors.New("xof: dst must not be empty")
	}
	h := hmac.New(sha256.New, seed[:])
	// » Length-prefix every variable-length input. Without it, (dst="ab",
	// » binder="c") and (dst="a", binder="bc") produce identical keys — a
	// » canonicalisation bug that silently destroys domain separation. This is
	// » the same class of mistake as unprefixed concatenation in a signature
	// » payload, and it shows up constantly in real protocols.
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], uint64(len(dst)))
	h.Write(n[:])
	h.Write(dst)
	binary.BigEndian.PutUint64(n[:], uint64(len(binder)))
	h.Write(n[:])
	h.Write(binder)

	x.key = h.Sum(nil)
	x.counter = 0
	x.buf = nil
	x.off = 0
	return nil
}

// Read implements XOF.
func (x *HMACCounter) Read(b []byte) (int, error) {
	if x.key == nil {
		return 0, errors.New("xof: Read before Init")
	}
	for n := 0; n < len(b); {
		if x.off >= len(x.buf) {
			h := hmac.New(sha256.New, x.key)
			var ctr [8]byte
			binary.BigEndian.PutUint64(ctr[:], x.counter)
			h.Write(ctr[:])
			x.buf = h.Sum(nil)
			x.off = 0
			x.counter++
		}
		copied := copy(b[n:], x.buf[x.off:])
		x.off += copied
		n += copied
	}
	return len(b), nil
}

// NextField64 implements XOF.
//
// TODO(week2): implement« — see EXERCISE 16».
func (x *HMACCounter) NextField64() (field.Field64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 16: uniform field elements, done correctly ────────────────
	// The obvious implementation is wrong and the bug is invisible:
	//
	//     read 8 bytes → uint64 → v % p          ← BIASED, do not do this
	//
	// 2^64 is not a multiple of p, so the low residues are very slightly more
	// likely. The bias here is about 2^-32 per sample, which no test you would
	// write by hand will notice, and which nonetheless means the share is not
	// uniformly distributed — and "the mask is uniform" is the entire security
	// argument for additive secret sharing (see internal/vdaf/field/field.go).
	//
	// Task: rejection sampling. Read 8 bytes little-endian; if the value is
	//       ≥ p, DISCARD it and read 8 more. Expected number of draws is
	//       2^64/p ≈ 1.0000000002, so the cost is nil.
	// Then: FillField64 must consume the stream in exactly the same order the
	//       draft specifies, because the client and the helper both derive the
	//       same share from the same seed and any divergence makes shares that
	//       do not sum to the input — which surfaces as "the aggregate is
	//       garbage" with no other symptom.
	// Spec: VDAF draft §6.1, the field element sampling procedure.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0, ErrTODO
}

// FillField64 implements XOF.
//
// TODO(week2): implement — repeated NextField64«, see EXERCISE 16».
func (x *HMACCounter) FillField64(out []field.Field64) error {
	return ErrTODO
}

// ─── helpers used across the Prio3 implementation ────────────────────────────

// Derive expands seed into n bytes with the given domain separator and binder.
func Derive(x XOF, seed Seed, dst, binder []byte, n int) ([]byte, error) {
	if err := x.Init(seed, dst, binder); err != nil {
		return nil, err
	}
	out := make([]byte, n)
	if _, err := x.Read(out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeriveSeed expands seed into another seed — the "seed to seed" step Prio3
// uses when deriving per-aggregator randomness from a single client seed.
func DeriveSeed(x XOF, seed Seed, dst, binder []byte) (Seed, error) {
	var out Seed
	b, err := Derive(x, seed, dst, binder, SeedSize)
	if err != nil {
		return out, err
	}
	copy(out[:], b)
	return out, nil
}
