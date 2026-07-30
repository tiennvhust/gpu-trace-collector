package privacy

import (
	"fmt"

	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// » THE ENCODER IS WHERE THE PRIVACY GUARANTEE IS EITHER TRUE OR A LIE, and it is
// » the least glamorous file in the project. internal/dp computes noise for a
// » declared sensitivity; internal/vdaf proves the client stayed inside a declared
// » bound. Both take the bound as GIVEN. This file is where the bound is actually
// » applied to real data, and if it is applied wrongly — or not at all — every ε in
// » the config is fiction while every test still passes.
// »
// » THREE RULES, and each one exists because breaking it is easy and invisible:
// »
// »   1. CLAMP BEFORE ENCODING. A datapoint outside [lo, hi] must be clamped, and
// »      the clamp must be the same bound the VDAF circuit enforces and the same
// »      one internal/dp used to compute sensitivity. Three places, one number:
// »      derive it from one source (the Task) rather than configuring it three
// »      times. See prio3« EXERCISE 32» for what happens when they diverge.
// »
// »   2. BOUND CONTRIBUTIONS PER DEVICE PER BATCH. A device that reports 60 times
// »      in a batch contributes 60× the sensitivity you calculated. Either enforce
// »      one report per device per batch, or multiply the sensitivity by the
// »      maximum and accept the accuracy loss. There is no third option, and
// »      "devices probably only report once" is not an enforcement mechanism.
// »
// »   3. NEVER PUT AN IDENTIFIER IN A MEASUREMENT. Not the hostname, not the PID,
// »      not the tenant name, not a hash of any of them. The measurement is a
// »      number in a bounded range and nothing else. A "just for debugging" device
// »      ID smuggled into a spare histogram bucket defeats the entire protocol,
// »      and it will be added by someone who does not realise that.

// Bucketing maps a continuous OTLP value onto a histogram bucket index.
type Bucketing interface {
	// Bucket returns the index for v, and false if v cannot be bucketed.
	Bucket(v float64) (int, bool)

	// Buckets is the number of buckets.
	Buckets() int
}

// LinearBuckets divides [Lo, Hi] into N equal buckets. Values outside the range
// are clamped into the end buckets.
//
// » Clamped, not dropped, and the distinction matters more than it looks. Dropping
// » out-of-range values makes participation depend on the VALUE, which biases the
// » aggregate and — worse — leaks: a device whose report is missing tells you its
// » value was out of range. Clamping keeps every device contributing exactly once,
// » which is what the sensitivity analysis assumes. Silent selection on the
// » sensitive attribute is a classic way to leak through an otherwise sound
// » mechanism.
type LinearBuckets struct {
	Lo, Hi float64
	N      int
}

// Bucket implements Bucketing.
//
// TODO(week4): implement.
func (b LinearBuckets) Bucket(v float64) (int, bool) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 61: bucketing, with the edges done properly ───────────────
	// Task: clamp v into [Lo, Hi], then map linearly onto [0, N).
	//
	// The cases that will bite you, in order of how quietly they fail:
	//   - v exactly == Hi must land in bucket N−1, not N. An index of N is an
	//     out-of-range write into the one-hot vector, which is either a panic or a
	//     corrupted report depending on how you wrote Encode.
	//   - NaN. Comparisons with NaN are all false, so a clamp written as
	//     `if v < Lo` then `if v > Hi` passes NaN straight through to
	//     int(NaN) — which is implementation-defined garbage. Reject NaN and Inf
	//     explicitly, return false, and count it. GPU telemetry produces NaN more
	//     often than you would like (a divide by a zero-length sample window).
	//   - Lo == Hi, or N <= 0: configuration errors that must fail at startup, not
	//     per datapoint.
	//
	// Then the design question for docs/PRIVACY.md: linear or logarithmic buckets
	// for kernel duration? A kernel takes anywhere from 2 µs to 200 ms, so linear
	// buckets put 99% of the mass in bucket 0. Log buckets fix that — and then ask
	// what DP noise of ±3 does to a log bucket whose true count is 5, versus one
	// whose true count is 50,000. The right bucketing depends on where you need
	// resolution, and DP makes that trade explicit in a way plaintext telemetry
	// never does. That last sentence is a good thing to be able to say.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0, false
}

// Buckets implements Bucketing.
func (b LinearBuckets) Buckets() int { return b.N }

// LogBuckets divides [Lo, Hi] into N logarithmically spaced buckets. Lo must be
// positive.
//
// » The right shape for latencies and durations, for the reason in« EXERCISE 61».
// » Note the family resemblance to OTLP's own ExponentialHistogram, which uses a
// » base-2 scale — worth reading, and worth considering whether to match its
// » bucket boundaries so the private and plaintext paths are comparable. If they
// » do not match, nobody can check one against the other, which is exactly the
// » validation you want during rollout.
type LogBuckets struct {
	Lo, Hi float64
	N      int
}

// Bucket implements Bucketing.
//
// TODO(week4): implement.
func (b LogBuckets) Bucket(v float64) (int, bool) { return 0, false }

// Buckets implements Bucketing.
func (b LogBuckets) Buckets() int { return b.N }

// Encoder turns OTLP datapoints into VDAF measurements for one task.
type Encoder struct {
	spec MetricSpec
}

// MetricSpec says which OTLP metric feeds a task, and how to turn it into a
// measurement.
type MetricSpec struct {
	// MetricName is the OTLP metric this task consumes, e.g. "gpu.sm.utilization".
	MetricName string

	// Kind selects the measurement shape.
	Kind MeasurementKind

	// Bucketing is used when Kind is KindHistogram.
	Bucketing Bucketing

	// MaxValue is the contribution bound when Kind is KindSum. It must equal the
	// bound the VDAF circuit enforces — see rule 1 at the top of this file.
	MaxValue uint64

	// Threshold is the value above which a KindCount measurement is true.
	Threshold float64
}

// MeasurementKind selects the VDAF variant a metric maps onto.
type MeasurementKind int

// Measurement kinds.
const (
	KindCount MeasurementKind = iota
	KindSum
	KindHistogram
)

// NewEncoder builds an encoder for one metric spec.
//
// TODO(week4): implement — validate the spec, and reject a spec whose Bucketing
// count does not match the task's VDAF output length. That mismatch produces
// reports the aggregators reject, en masse, with no obvious cause.
func NewEncoder(spec MetricSpec) (*Encoder, error) { return nil, ErrTODO }

// Encode extracts measurements from an OTLP metric.
//
// TODO(week4): implement.
func (e *Encoder) Encode(m *metricspb.Metric) ([]any, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 62: OTLP → measurements ───────────────────────────────────
	// Walk the metric's datapoints (see countDataPoints in internal/server/otlp.go
	// for the OTLP nesting) and produce one measurement per datapoint according to
	// e.spec.Kind:
	//   KindCount     → bool, datapoint value > Threshold
	//   KindSum       → uint64, clamped to [0, MaxValue]
	//   KindHistogram → int, e.spec.Bucketing.Bucket(value)
	//
	// THE ATTRIBUTES ARE THE TRAP, and it is the most likely way you accidentally
	// break the privacy property in this whole project. An OTLP datapoint carries
	// attributes: gpu.index, gpu.uuid, host.name, process.pid. THROW THEM ALL AWAY.
	// They are identifiers, and rule 3 at the top of this file says why.
	//
	// Which raises a real problem you have to solve rather than dodge: a host with
	// eight GPUs produces eight datapoints per export, so it contributes eight
	// times per batch and blows the contribution bound in rule 2. Options:
	//   (a) one report per HOST, aggregating the GPUs locally first (mean? max?
	//       — and note that choosing changes what the fleet aggregate means);
	//   (b) one report per GPU, and multiply the sensitivity by 8, accepting the
	//       accuracy loss;
	//   (c) sample one GPU per host per batch, uniformly at random.
	//
	// Pick one, implement it, and write the reasoning in docs/PRIVACY.md. There is
	// no free option, which is exactly why it is worth writing down — and (c) is
	// more interesting than it looks, because it is unbiased and costs nothing in
	// sensitivity.
	//
	// Also: count and export the datapoints you DROP, by reason. Silent drops in a
	// telemetry pipeline are how you find out three weeks later that a metric was
	// never being collected.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return nil, ErrTODO
}

// datapointValue extracts the numeric value from an OTLP number datapoint,
// which may be a double or an int64.
//
// TODO(week4): implement — handle both AsDouble and AsInt, and return false for a
// datapoint with neither set rather than silently treating it as zero. A zero that
// should have been "no data" biases the aggregate downward, which is the kind of
// bug that survives for years because the numbers look plausible.
func datapointValue(dp *metricspb.NumberDataPoint) (float64, bool) {
	return 0, false
}

// validateSpec checks a metric spec for internal consistency.
//
// TODO(week4): implement.
func validateSpec(s MetricSpec) error {
	if s.MetricName == "" {
		return fmt.Errorf("privacy: metric_name is required")
	}
	return ErrTODO
}
