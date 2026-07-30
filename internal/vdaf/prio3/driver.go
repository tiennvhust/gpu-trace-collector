package prio3

import (
	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/field"
	"github.com/tiennvhust/gpu-trace-collector/internal/vdaf/xof"
)

// driver implements VDAF generically over any FLP. Prio3Count, Prio3Sum and
// Prio3Histogram are the same protocol with different validity circuits.
//
// » Notice what this buys: the aggregators in internal/dap never learn which
// » variant they are running. A new metric shape means a new FLP and zero
// » changes to the aggregation, storage or collection code. That is the same
// » separation the plaintext path already has — internal/pipeline does not know
// » what OTLP is — and it is why "add a metric" is a config change rather than a
// » deploy of two services run by two organisations. On a real PPM deployment
// » that distinction is the difference between shipping a metric this week and
// » next quarter. (See also draft-ietf-ppm-dap-taskprov, the Apple-co-authored
// » extension that makes task provisioning itself a protocol rather than a
// » config-file exchange between two companies — this is the problem it solves.)
type driver struct {
	name  string
	algID uint32
	flp   FLP
	newX  func() xof.XOF
}

// Name implements VDAF.
func (d *driver) Name() string { return d.name }

// AlgorithmID implements VDAF.
func (d *driver) AlgorithmID() uint32 { return d.algID }

// RandSize is the number of bytes of client randomness Shard needs.
//
// TODO(week2): implement — derive from the FLP's ProveRandLen and the seeds.
func (d *driver) RandSize() int { return 0 }

// Shard implements VDAF.
//
// TODO(week2): implement.
func (d *driver) Shard(measurement any, nonce, rand []byte) ([]byte, [][]byte, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 24: Prio3 sharding, end to end ────────────────────────────
	// This is the client. Every device runs it, so its cost is the "privacy tax"
	// you are going to measure.
	//
	// Steps (VDAF draft §7.2.1 — follow the draft's order exactly, the
	// randomness stream is consumed positionally and getting the order wrong
	// fails the vectors while producing no error):
	//   1. encoded, err := d.flp.Encode(measurement)
	//   2. carve seeds out of `rand`: one per helper share, one for the prover
	//      randomness, one blind per aggregator for the joint randomness.
	//   3. shares := shardEncoded(...)                          [EXERCISE 19]
	//   4. jointRand := jointRandomness(...)                    [EXERCISE 20]
	//   5. proof := d.flp.Prove(encoded, proveRand, jointRand)
	//   6. shard the proof additively the same way as the input
	//   7. serialise: leader gets its full input+proof share; the helper gets a
	//      seed. Public share carries the joint-randomness parts.
	//
	// Validate `len(rand) == d.RandSize()` and REJECT rather than padding. A
	// short randomness buffer silently produces a report with a predictable
	// share, which is a total privacy failure that no test will show you.
	//
	// BENCHMARK IT (deliverable): ns/op and allocs/op for each variant, plus the
	// serialised report size. Then convert: at one report per device per hour on
	// 10^9 devices, what is the fleet-wide CPU cost, and what would it cost in
	// battery on a phone? That last number is the one nobody else in the
	// interview pipeline will have — you have six years of milliwatt budgets, so
	// use them.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, nil, ErrTODO
}

// prepState is the driver's PrepState.
type prepState struct {
	outputShare   OutputShare
	verifierPart  []field.Field64
	jointRandSeed xof.Seed
}

// Size implements PrepState.
func (s *prepState) Size() int {
	return len(s.outputShare)*field.EncodedSize64 +
		len(s.verifierPart)*field.EncodedSize64 + xof.SeedSize
}

// PrepInit implements VDAF.
//
// TODO(week2): implement.
func (d *driver) PrepInit(verifyKey []byte, aggID int, nonce, publicShare, inputShare []byte) (PrepState, []byte, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 25: the aggregator's half of verification ─────────────────
	// Steps (draft §7.2.2):
	//   1. decode this aggregator's input and proof share — expanding from the
	//      seed if aggID != LeaderID;
	//   2. derive query randomness from verifyKey, nonce and aggID. verifyKey is
	//      SHARED SECRET STATE between the two aggregators, established
	//      out of band per task. Understand why it must be secret from clients:
	//      a client that knows the challenge point can forge a proof that
	//      verifies, and robustness is gone. Write that in THREAT_MODEL.md, and
	//      note what an accidental verifyKey leak into a log would cost.
	//   3. verifierPart := d.flp.Query(inputShare, proofShare, queryRand, jointRand, Shares)
	//   4. outputShare := d.flp.Truncate(inputShare)
	//   5. return the state plus the prep share (the verifier part, and this
	//      aggregator's joint-randomness part so the peer can check it).
	//
	// Note step 4 happens BEFORE validity is known. That is deliberate — it
	// keeps the round count at one — but it means you must not aggregate the
	// output share until PrepNext accepts. Getting that ordering wrong lets
	// invalid reports into the aggregate, which is precisely the attack Prio
	// exists to stop, and it is an easy mistake to make when wiring
	// internal/dap/leader.go. Put a test on it.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, nil, ErrTODO
}

// PrepSharesToPrep implements VDAF.
//
// TODO(week2): implement — sum the verifier parts, check the joint-randomness
// parts agree, and emit the broadcast message.
func (d *driver) PrepSharesToPrep(prepShares [][]byte) ([]byte, error) {
	return nil, ErrTODO
}

// PrepNext implements VDAF.
//
// TODO(week2): implement — Decide on the combined verifier; return
// ErrInvalidReport when it rejects.
func (d *driver) PrepNext(state PrepState, prepMsg []byte) (OutputShare, error) {
	return nil, ErrTODO
}

// AggregateInit implements VDAF.
func (d *driver) AggregateInit() AggShare {
	return make(AggShare, d.flp.OutputLen())
}

// Unshard implements VDAF.
//
// TODO(week2): implement — sum the aggregate shares element-wise, then
// d.flp.Decode.
func (d *driver) Unshard(aggShares []AggShare, numMeasurements int) (any, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 26: unsharding, and the line where DP belongs ─────────────
	// Task: sum the shares in the field, then call d.flp.Decode.
	//
	// Then stop and notice where you are. The value you just computed is the
	// EXACT aggregate. Publishing it is a privacy failure even though no
	// aggregator ever saw a single report:
	//
	//   Collect the exact sum over batch B, then over B plus one more device.
	//   Subtract. You have that one device's contribution, exactly. Secret
	//   sharing did nothing to stop this, because the attack is against the
	//   published RESULT, not against the protocol.
	//
	// Two defences, and you need both:
	//   1. DP noise before publication (internal/dp) — bounds what any sequence
	//      of published aggregates reveals.
	//   2. Batch discipline — a minimum batch size, and no overlapping batches
	//      for the same task. See internal/dap/batch.go. DAP specifies these as
	//      protocol requirements precisely because the differencing attack above
	//      is so easy.
	//
	// Deliberate design decision: this function returns the exact value, and
	// noise is added by the COLLECTOR role in internal/dap/collector.go. Write
	// down why in docs/PRIVACY.md — the reason is that Unshard is also used in
	// tests and by the aggregators' own consistency checks, and a function that
	// silently randomises its output is untestable. Then make sure there is
	// exactly ONE code path from Unshard to a published result, and that it goes
	// through the noise. An alternative path is a privacy bug waiting for a
	// refactor.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}
