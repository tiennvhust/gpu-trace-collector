package dap

import (
	"bytes"
	"testing"
	"time"
)

func TestEncoderBigEndian(t *testing.T) {
	// » Pin the byte order. DAP is big-endian; VDAF field elements are
	// » little-endian. Both live in the same message, and one known-answer test in
	// » each package is what stops you burning an afternoon on it.
	e := &Encoder{}
	e.U16(0x0102)
	e.U32(0x03040506)
	if want := []byte{1, 2, 3, 4, 5, 6}; !bytes.Equal(e.B, want) {
		t.Fatalf("encoded %v, want %v (big-endian)", e.B, want)
	}
}

func TestVec16RoundTrip(t *testing.T) {
	e := &Encoder{}
	payload := []byte("hello")
	if err := e.Vec16(payload); err != nil {
		t.Fatalf("Vec16: %v", err)
	}
	if len(e.B) != 2+len(payload) {
		t.Fatalf("encoded %d bytes, want %d", len(e.B), 2+len(payload))
	}

	d := NewDecoder(e.B)
	got, err := d.Vec16()
	if err != nil {
		t.Fatalf("Vec16 decode: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("got %q, want %q", got, payload)
	}
	if !d.Done() {
		t.Errorf("%d bytes left unconsumed", d.Remaining())
	}
}

func TestVec16RejectsOversizedPayload(t *testing.T) {
	e := &Encoder{}
	if err := e.Vec16(make([]byte, 65536)); err == nil {
		t.Error("Vec16 accepted a payload that cannot fit a uint16 length: silent truncation")
	}
}

func TestDecoderRejectsTruncatedInput(t *testing.T) {
	// » Every one of these is a potential panic in an HTTP handler on a public
	// » endpoint, which is a remote denial of service against the whole task.
	// » Bounds-check on every read.
	cases := []struct {
		name string
		in   []byte
		read func(*Decoder) error
	}{
		{"u16 on 1 byte", []byte{1}, func(d *Decoder) error { _, err := d.U16(); return err }},
		{"u32 on 3 bytes", []byte{1, 2, 3}, func(d *Decoder) error { _, err := d.U32(); return err }},
		{"u64 on 7 bytes", make([]byte, 7), func(d *Decoder) error { _, err := d.U64(); return err }},
		{"id on 15 bytes", make([]byte, 15), func(d *Decoder) error { _, err := d.ID(); return err }},
		{"vec16 length only", []byte{0, 10}, func(d *Decoder) error { _, err := d.Vec16(); return err }},
		{"vec16 short body", []byte{0, 10, 1, 2}, func(d *Decoder) error { _, err := d.Vec16(); return err }},
		{"empty", nil, func(d *Decoder) error { _, err := d.U8(); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.read(NewDecoder(tc.in)); err == nil {
				t.Error("want an error, got nil")
			}
		})
	}
}

func TestVec32DoesNotTrustTheLengthPrefix(t *testing.T) {
	// » A 4-byte length prefix claiming 4 GiB on a 6-byte body. If Vec32 allocates
	// » before checking against Remaining(), one small request exhausts the
	// » leader's memory. Classic decompression-bomb shape, trivially remotely
	// » triggerable, and one comparison to prevent.
	d := NewDecoder([]byte{0xff, 0xff, 0xff, 0xff, 0x01, 0x02})
	if _, err := d.Vec32(); err == nil {
		t.Error("Vec32 accepted a length prefix far larger than the remaining input")
	}
}

func TestChecksumIsOrderIndependent(t *testing.T) {
	// » The property the protocol needs: the two aggregators process reports in
	// » whatever order they arrive, and must still agree. See« EXERCISE 40».
	a := ReportID{1}
	b := ReportID{2}
	c := ReportID{3}

	if Checksum([]ReportID{a, b, c}) != Checksum([]ReportID{c, a, b}) {
		t.Error("checksum depends on order: the two aggregators will never agree")
	}
	if Checksum([]ReportID{a, b}) == Checksum([]ReportID{a, c}) {
		t.Error("different report sets produced the same checksum")
	}
	if Checksum(nil) != ([32]byte{}) {
		t.Error("the empty batch should checksum to zero")
	}
}

func TestTimePrecisionRoundsDown(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	const precision = time.Hour

	got := TimePrecision(base, precision)
	if got.Unix()%int64(precision.Seconds()) != 0 {
		t.Errorf("TimePrecision(%v) = %v, not a multiple of %v", base, got, precision)
	}
	if got.After(base) {
		t.Errorf("TimePrecision rounded up: %v > %v", got, base)
	}

	// » Idempotence: rounding an already-rounded time must not move it. Without
	// » this, a report re-timestamped on retry can land in a different batch, and a
	// » report in two batches is the differencing attack.
	if again := TimePrecision(got, precision); !again.Equal(got) {
		t.Errorf("TimePrecision is not idempotent: %v → %v", got, again)
	}
}

func TestIntervalIsHalfOpen(t *testing.T) {
	start := time.Unix(1000, 0).UTC()
	i := Interval{Start: start, Duration: 100 * time.Second}

	if !i.Contains(start) {
		t.Error("the start instant must be inside the interval")
	}
	if i.Contains(start.Add(100 * time.Second)) {
		t.Error("the end instant must be outside the interval (half-open)")
	}
	if i.Contains(start.Add(-time.Nanosecond)) {
		t.Error("an instant before the start must be outside")
	}
}

func TestParseTaskID(t *testing.T) {
	id, err := ParseTaskID("000102030405060708090a0b0c0d0e0f")
	if err != nil {
		t.Fatalf("ParseTaskID: %v", err)
	}
	if id[0] != 0x00 || id[15] != 0x0f {
		t.Errorf("parsed %v, want 00..0f", id)
	}
	for _, bad := range []string{"", "zz", "0001", "000102030405060708090a0b0c0d0e0f00"} {
		if _, err := ParseTaskID(bad); err == nil {
			t.Errorf("ParseTaskID(%q) should fail", bad)
		}
	}
}
