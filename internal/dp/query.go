package dp

// » THE HARD PART OF DP IS NOT THE NOISE. Sampling Laplace is ten lines.
// » Establishing that your query really has the sensitivity you claim is where
// » real deployments break, and it is the part that requires knowing the data.
// »
// » Sensitivity is a property of the QUERY plus the CONTRIBUTION BOUND, not of
// » the dataset. "Sum of GPU utilisation" has unbounded sensitivity — a single
// » device reporting 10^9 moves the sum by 10^9, so no finite noise hides it.
// » You make it finite by clamping the per-contribution value to [lo, hi] and by
// » bounding how many rows one device may contribute. Both bounds are policy
// » decisions with an accuracy cost, and both must be enforced where the
// » adversary cannot skip them.
// »
// » Which is precisely why the private path enforces them in the VDAF circuit
// » (internal/vdaf/prio3) rather than in a server-side clamp: a malicious client
// » would simply not clamp. Prio's verifiability exists so that "the client
// » clamped its input" is something the aggregators can CHECK without seeing
// » the input. Hold on to that sentence — it is the answer to "why does Prio
// » need verifiability?" in §7 of the plan.

// Query describes an aggregation whose sensitivity is known by construction.
type Query interface {
	// Sensitivity returns (L1, L2) sensitivity for one contributor.
	Sensitivity() (l1, l2 float64)
}

// Count counts contributors. Each device contributes at most one increment.
//
// » L1 = L2 = 1, the easiest case in DP and a good sanity anchor: whatever else
// » you build, a count at ε = 1 should have noise standard deviation ≈ √2 ≈ 1.41
// » under Laplace. If your histogram's per-bucket error is wildly different from
// » that, your sensitivity accounting is wrong, not your sampler.
type Count struct{}

// Sensitivity implements Query.
func (Count) Sensitivity() (float64, float64) { return 1, 1 }

// BoundedSum sums a per-contributor value clamped to [Lo, Hi], with at most
// MaxContributions rows per contributor.
type BoundedSum struct {
	Lo, Hi           float64
	MaxContributions int
}

// Sensitivity implements Query.
//
// TODO(week1): implement.
func (b BoundedSum) Sensitivity() (float64, float64) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 4: bounded-sum sensitivity ────────────────────────────────
	// Task: return the correct L1 and L2 sensitivity for this query.
	//
	// The trap: it is NOT simply Hi. Think about what "one contributor is
	// added or removed" does to the sum when Lo is negative — a device
	// contributing Lo and a device contributing Hi are both one removal away
	// from the neighbouring dataset, so the worst-case swing is max(|Lo|, |Hi|),
	// times MaxContributions. Under the *bounded*-DP definition (a row is
	// replaced rather than removed) the answer is Hi − Lo instead. State which
	// definition you are using in docs/PRIVACY.md, because the two differ by 2×
	// and someone will ask you which one you meant.
	//
	// Then: with MaxContributions > 1, is L2 = √k · max(|Lo|,|Hi|) or
	// k · max(|Lo|,|Hi|)? Sketch the vector of one user's contributions to see
	// which. This is the same reasoning that makes Gaussian beat Laplace for
	// gradients in Project B.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0, 0
}

// Clamp truncates v into [Lo, Hi]. Applying this is what MAKES the sensitivity
// bound true; skipping it makes Sensitivity a lie.
//
// TODO(week1): implement.
func (b BoundedSum) Clamp(v float64) float64 {
	return 0
}

// Histogram counts contributors into Buckets disjoint buckets, where each
// contributor falls into at most MaxBuckets of them.
//
// » The interesting question for this project: your GPU telemetry has, say, 64
// » utilisation buckets and 10⁴ reporting devices per collection. At ε = 2 with
// » a discrete Gaussian, what is the per-bucket standard deviation, and what is
// » the smallest bucket count you can distinguish from zero with 95%
// » confidence? Work it out numerically in histogram_test.go and write the
// » answer in docs/BENCHMARKS.md. Being able to say "at ε = 2 and n = 10⁴ I can
// » resolve buckets down to about X" is the difference between having read
// » about DP and having used it.
type Histogram struct {
	Buckets    int
	MaxBuckets int
}

// Sensitivity implements Query.
//
// TODO(week1): implement — one contributor touching MaxBuckets buckets by 1 each.
func (h Histogram) Sensitivity() (float64, float64) {
	return 0, 0
}

// PerBucketStdDev returns the standard deviation of the noise each bucket of a
// histogram receives under mech. Use it to answer "is this ε usable?" before
// deploying a task, not after.
//
// TODO(week1): implement.
func (h Histogram) PerBucketStdDev(mech Mechanism) (float64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 5: make the accuracy question answerable in code ───────────
	// Task: given the mechanism's Params and this histogram's sensitivity,
	//       return the per-bucket noise standard deviation (√2·b for Laplace
	//       with scale b; σ for Gaussian). Type-switch on mech, or better, add
	//       a StdDev(sensitivity) method to the Mechanism interface and let each
	//       mechanism answer for itself — the second design is the one you want
	//       when you add a third mechanism.
	// Then wire this into internal/privacy task validation so that configuring
	//       a task whose expected per-bucket error exceeds, say, 10% of the
	//       expected bucket count FAILS AT STARTUP with a readable message.
	//       Refusing to deploy a privacy setting that cannot produce usable
	//       data is an operational feature, and it is the kind of thing that
	//       makes a portfolio project read as production work.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0, ErrTODO
}
