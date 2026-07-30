// Package dp implements the differential-privacy layer of the private
// aggregation path: noise mechanisms, sensitivity-aware queries, and a Rényi-DP
// accountant that tracks how much of a task's privacy budget is spent.
//
// Secret sharing (internal/vdaf) and differential privacy are stacked because
// neither suffices alone. Secret sharing hides each report from each aggregator
// but publishes an exact aggregate, and exact aggregates over two overlapping
// batches subtract to one device's contribution. DP bounds what any published
// aggregate reveals, but adding the noise requires someone to hold the raw
// values — unless the aggregate was computed under secret sharing first.
//
// » THE PARAGRAPH ABOVE IS THE SINGLE MOST IMPORTANT IDEA IN THE PROJECT, and it
// » is worth being able to say out loud in thirty seconds. Note what each half
// » defends against:
// »
// »   SECRET SHARING defends against the AGGREGATOR. It says nothing about what
// »   the published number reveals.
// »   DIFFERENTIAL PRIVACY defends against whoever reads the PUBLISHED NUMBERS,
// »   including an adversary who sees every aggregate you ever release and
// »   already knows every other user's value. It says nothing about who held the
// »   raw data.
// »
// » Different adversaries, different mechanisms, and §5 of THREAT_MODEL.md is
// » where you write down which party is trusted for what.
//
// » MENTAL MODEL FOR ε. A mechanism M is (ε, δ)-DP if for any two datasets D,
// » D′ differing in one user's data, and any output set S:
// »
// »     Pr[M(D) ∈ S] ≤ e^ε · Pr[M(D′) ∈ S] + δ
// »
// » Read it as a bound on the likelihood ratio an attacker can achieve when
// » testing "was Alice in the dataset?". ε = 0 means the output is independent
// » of Alice entirely (useless but perfectly private). ε = 1 means the
// » attacker's odds shift by at most e ≈ 2.7×. δ is the probability the bound
// » fails outright, so it must be far below 1/n — a δ of 10⁻³ with a million
// » users licenses a mechanism that publishes ~1000 users verbatim.
//
// » Reading, in this order:
// »   Near & Abuah, Programming Differential Privacy, ch. 1–8 — programmingdp.com
// »   Dwork & Roth, The Algorithmic Foundations of DP, ch. 2–3 only
// »   Apple, "Learning with Privacy at Scale" (local model, in production)
// »   Canonne, Kamath & Steinke, "The Discrete Gaussian for DP" (arXiv 2004.00010)
package dp

import "errors"

// ErrTODO marks a function that is not implemented yet.
//
// » Every exercise in this package returns it until you replace the body, so a
// » failing test that reports ErrTODO is telling you which exercise to do next.
var ErrTODO = errors.New("dp: not implemented«, see the EXERCISE block above this function»")

// Params is an (ε, δ) differential-privacy guarantee.
//
// » Delta == 0 is "pure" ε-DP, achievable with the Laplace mechanism. Delta > 0
// » is "approximate" DP, which the Gaussian mechanism needs and which composes
// » much more gracefully over many releases — the reason essentially every
// » production deployment lives here.
type Params struct {
	Epsilon float64
	Delta   float64
}

// Valid reports whether the parameters are in range: ε > 0 and δ ∈ [0, 1).
func (p Params) Valid() bool {
	return p.Epsilon > 0 && p.Delta >= 0 && p.Delta < 1
}

// Mechanism perturbs an already-aggregated value so that the result is
// differentially private with respect to any single contributor.
//
// » Note the signature takes the aggregate, not the dataset: this is the
// » CENTRAL model, where noise is added once at aggregation time. Compare:
// »
// »   LOCAL     — every device noises its own value before sending. No trusted
// »               party at all, but error grows as √n instead of staying O(1),
// »               so you need enormous n. Apple's emoji/QuickType telemetry.
// »   CENTRAL   — one trusted curator holds raw data and noises the aggregate.
// »               Best accuracy, worst trust assumption.
// »   SHUFFLE   — devices add a little noise, an oblivious shuffler strips
// »               identity, the server aggregates. Accuracy near central with
// »               a much weaker trust assumption. Prio/DAP is a cousin of this
// »               idea: two non-colluding aggregators replace the shuffler, so
// »               "central-model noise" becomes safe to add without any single
// »               party ever seeing a raw value.
// »
// » Being able to place a given deployment in that taxonomy, and defend the
// » choice, is a guaranteed interview question. See §7 of the plan.
type Mechanism interface {
	// Apply returns the noised aggregate. sensitivity is the maximum amount
	// the true aggregate can change when one contributor is added or removed.
	Apply(aggregate, sensitivity float64) (float64, error)

	// Params returns the guarantee this mechanism provides per application.
	Params() Params
}
