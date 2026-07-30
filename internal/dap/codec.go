package dap

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// » DAP messages use the TLS presentation language (RFC 8446 §3), not JSON or
// » protobuf. Fixed-width big-endian integers, and variable-length vectors
// » prefixed with a length whose own width is chosen to fit the maximum:
// »
// »     opaque extensions<0..2^16-1>;   →  2-byte length, then that many bytes
// »
// » WHY, given that this repository already speaks protobuf everywhere else? Two
// » reasons, and the second is the interesting one:
// »
// »   1. Canonical encoding. Every message has exactly one valid byte
// »      representation. Protobuf does not: field order, varint padding and
// »      unknown fields all admit variation. For messages that are HASHED or
// »      COMPARED — and the report ID's anti-replay check and the batch
// »      "checksum" both are — non-canonical encodings mean two byte strings that
// »      are semantically identical and hash differently. That is not a style
// »      preference, it is a correctness requirement.
// »   2. No unknown fields. Protobuf's forward compatibility is a feature
// »      everywhere else and a liability here: an aggregator MUST NOT silently
// »      ignore a field it does not understand, because the field might be the
// »      one that changes what the message means. DAP wants strict, total parsing
// »      of a fixed grammar. Fail closed.
// »
// » Note the contrast with the plaintext path, which forwards OTLP protobuf
// » verbatim precisely BECAUSE protobuf tolerates schema skew. Same codebase, two
// » encodings, opposite requirements, and being able to explain why is worth more
// » than either fact alone.
// »
// » AND THE ENDIANNESS TRAP: DAP is big-endian (network order); VDAF field
// » elements are little-endian. Both, in the same message. See the note in
// » internal/vdaf/field/field.go.

// ErrShortBuffer is returned when a decode runs past the end of its input.
var ErrShortBuffer = errors.New("dap: truncated message")

// Encoder appends values to a growing byte slice.
//
// » Append-only with no intermediate allocation, because the leader encodes one
// » of these per report at ingest rate. Reuse the buffer across reports with a
// » sync.Pool once you get to the load test — it is one of the two or three
// » changes that actually move the number.
type Encoder struct{ B []byte }

// U8 appends a single byte.
func (e *Encoder) U8(v uint8) { e.B = append(e.B, v) }

// U16 appends a big-endian uint16.
func (e *Encoder) U16(v uint16) { e.B = binary.BigEndian.AppendUint16(e.B, v) }

// U32 appends a big-endian uint32.
func (e *Encoder) U32(v uint32) { e.B = binary.BigEndian.AppendUint32(e.B, v) }

// U64 appends a big-endian uint64.
func (e *Encoder) U64(v uint64) { e.B = binary.BigEndian.AppendUint64(e.B, v) }

// Bytes appends raw bytes with no length prefix (fixed-length fields).
func (e *Encoder) Bytes(b []byte) { e.B = append(e.B, b...) }

// Vec16 appends b as opaque<0..2^16-1>.
//
// TODO(week3): implement.
func (e *Encoder) Vec16(b []byte) error {
	// EXERCISE-BEGIN
	// ─── EXERCISE 36: length-prefixed vectors ───────────────────────────────
	// Task: write len(b) as a big-endian uint16, then b. Return an error if
	//       len(b) > 65535 rather than truncating — a silent truncation here
	//       produces a message that decodes successfully into the wrong thing,
	//       which is the worst failure mode a codec can have.
	// Then add Vec24 and Vec32. DAP uses a 3-byte length in at least one place
	//       (check the draft), which is exactly the kind of detail that only
	//       shows up when you implement rather than read.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return ErrTODO
}

// Vec32 appends b as opaque<0..2^32-1>.
//
// TODO(week3): implement.
func (e *Encoder) Vec32(b []byte) error { return ErrTODO }

// Decoder reads values from a byte slice, tracking position.
//
// » Every method must be safe against a truncated or hostile input, because this
// » parses bytes from the network before any authentication has happened. One
// » missing bounds check is a panic in an HTTP handler, and a panic in the
// » leader's upload path is a remote denial of service against the whole task.
// » Bounds-check on every read, and fuzz it« — see EXERCISE 38».
type Decoder struct {
	b   []byte
	off int
}

// NewDecoder returns a decoder over b.
func NewDecoder(b []byte) *Decoder { return &Decoder{b: b} }

// Remaining reports how many bytes are unread.
func (d *Decoder) Remaining() int { return len(d.b) - d.off }

// Done reports whether the whole input was consumed.
//
// » Call this at the end of every top-level decode and REJECT trailing bytes. A
// » message with junk appended must not be accepted: if the leader accepts it and
// » the helper rejects it, the two aggregators disagree about which reports exist,
// » and the batch can never be collected. "Both aggregators must reach the same
// » decision about every report" is an invariant the codec has to help maintain.
func (d *Decoder) Done() bool { return d.off == len(d.b) }

// U8 reads one byte.
//
// TODO(week3): implement.
func (d *Decoder) U8() (uint8, error) { return 0, ErrTODO }

// U16 reads a big-endian uint16.
//
// TODO(week3): implement.
func (d *Decoder) U16() (uint16, error) { return 0, ErrTODO }

// U32 reads a big-endian uint32.
//
// TODO(week3): implement.
func (d *Decoder) U32() (uint32, error) { return 0, ErrTODO }

// U64 reads a big-endian uint64.
//
// TODO(week3): implement.
func (d *Decoder) U64() (uint64, error) { return 0, ErrTODO }

// Bytes reads exactly n bytes.
//
// TODO(week3): implement.
func (d *Decoder) Bytes(n int) ([]byte, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 37: the aliasing decision ─────────────────────────────────
	// Task: bounds-check, advance the offset, return the bytes.
	//
	// The decision to make consciously: return d.b[off:off+n] (a slice ALIASING
	// the input) or a copy? Aliasing is free and is what you want on the hot
	// path. But it means the caller holds a reference into the whole request
	// buffer, so a ReportID retained in the anti-replay map pins the entire
	// upload body in memory — a slow leak that looks like "the leader's RSS grows
	// under load" and takes a heap profile to find.
	//
	// Decide per call site, and document the choice on this method. The
	// convention that works: alias by default, and copy explicitly at the
	// boundaries where a value outlives the request (anything going into
	// internal/dap/store.go). Then put a comment at each of those copies saying
	// why, because the next reader will otherwise "optimise" it away.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// Vec16 reads an opaque<0..2^16-1>.
//
// TODO(week3): implement.
func (d *Decoder) Vec16() ([]byte, error) { return nil, ErrTODO }

// Vec32 reads an opaque<0..2^32-1>.
//
// TODO(week3): implement — and think about the allocation: a 2^32 length prefix
// on a 100-byte body must not make you allocate 4 GiB. Check the declared length
// against Remaining() BEFORE allocating anything. This is a classic
// decompression-bomb shape and it is trivially remotely triggerable.
func (d *Decoder) Vec32() ([]byte, error) { return nil, ErrTODO }

// ID reads a fixed-size 16-byte identifier.
//
// TODO(week3): implement.
func (d *Decoder) ID() ([IDLen]byte, error) {
	var id [IDLen]byte
	return id, ErrTODO
}

// errShort formats a truncation error with useful context.
//
// » "dap: truncated message" alone is unhelpful at 3am. Say what was being read,
// » how many bytes were wanted and how many were left — cheap to add, and the
// » difference between a five-minute diagnosis and an hour of adding printfs.
func errShort(what string, want, have int) error {
	return fmt.Errorf("%w: reading %s wanted %d bytes, %d remain", ErrShortBuffer, what, want, have)
}

// EXERCISE-BEGIN
// ─── EXERCISE 38: fuzz the codec ────────────────────────────────────────────
// This codec parses attacker-controlled bytes on a public endpoint. It is the
// single best fuzzing target in the repository, and Go makes it nearly free.
//
// Task: add codec_fuzz_test.go with
//
//	func FuzzDecodeReport(f *testing.F) {
//	    f.Add(validReportBytes)                  // seed corpus
//	    f.Fuzz(func(t *testing.T, data []byte) {
//	        r, err := DecodeReport(data)
//	        if err != nil { return }             // rejecting is fine
//	        // Round-trip invariant: anything that decodes must re-encode to
//	        // exactly the same bytes. This catches non-canonical encodings,
//	        // which matter because report IDs are hashed (see codec.go).
//	        got, err := r.Encode()
//	        if err != nil { t.Fatal(err) }
//	        if !bytes.Equal(got, data) { t.Fatalf("not canonical") }
//	    })
//	}
//
// Run: go test -fuzz=FuzzDecodeReport -fuzztime=5m ./internal/dap
//
// Expect it to find something in the first minute — probably an unbounded
// allocation from a length prefix, or a panic on an empty slice. Add the crashers
// to testdata/fuzz/ (Go does this automatically) so they stay regression tests.
//
// Then put a sentence in the README: "codec fuzzed with Go's native fuzzer;
// N crashers found and fixed." That is a concrete, verifiable security claim, and
// it costs you one afternoon.
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END
