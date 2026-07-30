package prio3

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/xof"
)

// » THE MOST VALUABLE FILE IN THE REPOSITORY. Everything else in this project is
// » something a reviewer has to take on trust. This is objective, third-party,
// » machine-checkable proof that you read an IETF specification correctly and
// » implemented it byte for byte. From the plan: "Passing published test vectors
// » is the single most credible artifact in this whole plan."
// »
// » Two rules for week 2:
// »   NEVER edit a vector to make a test pass. If they disagree, you are wrong.
// »   NEVER skip these to save time. They are the point.
// »
// » Get them green and put the command and its output in the README.

// vector mirrors the JSON layout of the draft's test vectors.
//
// » VERIFY THIS STRUCT AGAINST THE ACTUAL FILES rather than trusting the shape
// » here — the schema has changed across draft revisions (the `ctx` field is
// » recent; `agg_param` changed type; `prep` used to be flat). Print one file and
// » read it before you write any decoding logic. This is a five-minute step that
// » saves an hour of confusing failures.
type vector struct {
	Shares    int    `json:"shares"`
	VerifyKey string `json:"verify_key"`
	Ctx       string `json:"ctx"`

	// Variant parameters, present only for the variants that take them.
	Length         *int    `json:"length"`
	ChunkLength    *int    `json:"chunk_length"`
	MaxMeasurement *uint64 `json:"max_measurement"`
	Bits           *int    `json:"bits"`

	AggResult []uint64 `json:"agg_result"`
	AggShares []string `json:"agg_shares"`

	Prep []struct {
		Measurement  json.RawMessage `json:"measurement"`
		Nonce        string          `json:"nonce"`
		Rand         string          `json:"rand"`
		PublicShare  string          `json:"public_share"`
		InputShares  []string        `json:"input_shares"`
		PrepShares   [][]string      `json:"prep_shares"`
		PrepMessages []string        `json:"prep_messages"`
		OutShares    [][]string      `json:"out_shares"`
	} `json:"prep"`
}

func TestDraftTestVectors(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(paths) == 0 {
		// » A skip, not a failure: missing vectors are a download step, not a
		// » code defect. Read testdata/README.md for where to get them.
		t.Skip("no test vectors in testdata/ — see internal/vdaf/prio3/testdata/README.md")
	}

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			var v vector
			if err := json.Unmarshal(raw, &v); err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			if v.Shares != Shares {
				t.Skipf("vector uses %d shares, this implementation targets %d", v.Shares, Shares)
			}
			runVector(t, filepath.Base(path), v)
		})
	}
}

// runVector replays one vector through the implementation.
//
// TODO(week2): implement.
func runVector(t *testing.T, name string, v vector) {
	t.Helper()

	// EXERCISE-BEGIN
	// ─── EXERCISE 34: the test-vector harness ───────────────────────────────
	// Build this in the order below. Each step is a checkpoint you can debug in
	// isolation — do NOT write the whole thing and then run it, because a
	// mismatch at step 5 with no step-3 assertion is nearly undebuggable.
	//
	//   1. Pick the VDAF from the filename ("Prio3Count_0.json" → Prio3Count),
	//      using the vector's own Length / ChunkLength / MaxMeasurement fields
	//      rather than defaults of your own.
	//   2. Decode verify_key, nonce and rand from hex.
	//   3. Shard(measurement, nonce, rand) and compare public_share and
	//      input_shares byte for byte. STOP HERE until this passes for
	//      Prio3Count. Nothing downstream can be right if sharding is not, and
	//      almost every mismatch you will hit is in this step — usually the
	//      order in which `rand` is consumed, or the XOF's dst string.
	//   4. PrepInit per aggregator; compare prep_shares.
	//   5. PrepSharesToPrep; compare prep_messages.
	//   6. PrepNext; compare out_shares.
	//   7. Aggregate; compare agg_shares.
	//   8. Unshard; compare agg_result.
	//
	// DEBUGGING ADVICE, from the shape of the failure:
	//   - Sharding differs but is the right LENGTH → the XOF (dst, binder, or
	//     rejection sampling). Go back to EXERCISE 15/16 and the KAT.
	//   - Right length, right first element, diverges later → the randomness
	//     stream is being consumed in the wrong order somewhere.
	//   - Everything matches except agg_result → Decode, or numMeasurements.
	//   - Nothing matches at all → check `ctx` and the algorithm ID are in your
	//     domain separation. Recent drafts bind the application context into
	//     every derivation, and omitting it changes every byte.
	//
	// Report a byte-level diff on failure (offset, want, got) rather than
	// dumping two hex blobs. You will read this output many times.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	t.Fatalf("EXERCISE 34: implement runVector to replay %s", name)
}

// mustHex decodes a hex string from a vector, failing the test on bad input.
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

// newXOF returns the XOF the vectors require.
//
// » Must be TurboShake128 once« EXERCISE 15» is done. Until then this returns the
// » stand-in, and the vectors will fail on the very first byte comparison — that
// » is expected and is the honest state of the world, not a bug to work around.
func newXOF() xof.XOF { return &xof.HMACCounter{} }

func TestRoundTripWithoutVectors(t *testing.T) {
	// » Your own end-to-end test, independent of the draft: shard → prep →
	// » aggregate → unshard for a known set of measurements, and check the result
	// » is the arithmetic answer. This is what you iterate against while
	// » building, because it fails fast and its failures are legible. The vectors
	// » are the acceptance gate; this is the development loop.
	//
	// EXERCISE-BEGIN
	// ─── EXERCISE 35: your own end-to-end test ──────────────────────────────
	// Task: for Prio3Count with measurements [true, true, false, true], drive both
	//       aggregators in-process and assert the aggregate is 3. Then the same
	//       for Prio3Histogram with length 4 and measurements [0,1,1,3] → counts
	//       [1,2,0,1], and Prio3Sum with max 1000 and [10, 20, 30] → 60.
	// Then the negative cases: corrupt one input share and assert
	//       ErrInvalidReport rather than a wrong aggregate. A protocol that
	//       silently accepts corrupted reports is worse than one that crashes.
	// Table-drive it over the three variants so adding SumVec later is one row.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	t.Fatalf("EXERCISE 35: implement the in-process round trip")
}

func BenchmarkShard(b *testing.B) {
	// » Fill this in once Shard works. It produces the client-side number in
	// » docs/BENCHMARKS.md — the one that answers "what does this cost on a
	// » device". Report ns/op, B/op and allocs/op for Prio3Histogram at 16, 128
	// » and 1024 buckets, and note that allocs/op matters more than ns/op on a
	// » phone: allocation means GC, and GC means the radio stays awake longer.
	b.Skip("EXERCISE 24: implement Shard, then benchmark it")
}
