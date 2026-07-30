package dp

import (
	"fmt"
	"math"
	"sync"
)

// » WHY AN ACCOUNTANT EXISTS AT ALL. A privacy guarantee is not a property of
// » one release, it is a property of everything you have ever published about a
// » user. Publish the same noised count twice with independent noise and an
// » attacker averages the two, halving the effective noise. Publish it a
// » thousand times and the noise is gone. So ε is a BUDGET that is spent, and
// » something has to hold the ledger — otherwise "we're ε = 1" is a claim about
// » a single query in a system that runs continuously.
// »
// » This is also the source of the most interesting non-technical question in
// » §7 of the plan: the quarter's budget is spent and product wants one more
// » metric. The honest answers are (a) wait for the next period, (b) drop an
// » existing metric to free budget, (c) accept coarser data for everything by
// » re-noising at a lower ε, (d) change the collection so the metric needs less
// » budget (fewer buckets, longer aggregation window, larger minimum batch).
// » Notice that (d) is the engineering answer and the other three are policy.
// » Having a ledger in the code is what turns that conversation from a
// » negotiation into arithmetic.
// »
// » Composition, three tiers, worth knowing all three:
// »   BASIC     k releases at (εᵢ, δᵢ) compose to (Σεᵢ, Σδᵢ). Always true,
// »             always loose. Linear in k.
// »   ADVANCED  (Dwork–Rothblum–Vadhan) k releases at ε each give roughly
// »             ε√(2k·ln(1/δ′)) + kε(e^ε − 1) for any δ′ > 0. Sublinear: √k.
// »   RÉNYI     (Mironov, arXiv 1702.07476) track the Rényi divergence at
// »             several orders α; it adds EXACTLY under composition, and you
// »             convert to (ε, δ) once, at the end. This is what every serious
// »             library does, and it is what makes DP-SGD's thousands of steps
// »             accountable in Project B.

// Accountant tracks privacy loss across many releases and converts it to a
// single (ε, δ) statement.
type Accountant interface {
	// Add records one application of a Gaussian mechanism with the given noise
	// multiplier (σ / sensitivity) and subsampling rate q ∈ (0, 1]. q = 1 means
	// every contributor participates in every release.
	Add(noiseMultiplier, q float64, steps int) error

	// Total converts the accumulated loss to (ε, δ) at the requested δ.
	Total(delta float64) (float64, error)
}

// BasicAccountant composes by summing ε and δ. Correct, pessimistic, and a
// useful reference to check the RDP accountant against: RDP must never report
// a LARGER ε than this for the same sequence of releases.
type BasicAccountant struct {
	epsilon float64
	delta   float64
}

// AddParams records one release with an explicit (ε, δ).
func (a *BasicAccountant) AddParams(p Params) error {
	if !p.Valid() {
		return fmt.Errorf("dp: invalid params %+v", p)
	}
	a.epsilon += p.Epsilon
	a.delta += p.Delta
	return nil
}

// Add implements Accountant for Gaussian releases.
//
// TODO(week1): implement — convert (noiseMultiplier, q) to an (ε, δ) per step,
// then call AddParams steps times. Start with q = 1 and reject q < 1 until
//« EXERCISE 7» gives you subsampling amplification.
func (a *BasicAccountant) Add(noiseMultiplier, q float64, steps int) error {
	return ErrTODO
}

// Total implements Accountant.
func (a *BasicAccountant) Total(delta float64) (float64, error) {
	if delta < a.delta {
		return 0, fmt.Errorf("dp: requested delta %g below accumulated delta %g", delta, a.delta)
	}
	return a.epsilon, nil
}

// RDPAccountant tracks Rényi differential privacy at a fixed set of orders α.
//
// » The trick that makes this work: for a mechanism M, RDP at order α is
// »     D_α(M(D) ‖ M(D′)) ≤ ρ(α)
// » and for the Gaussian mechanism with noise multiplier σ this is exactly
// »     ρ(α) = α / (2σ²)
// » — no approximation, no numeric search. Composition is then just addition of
// » ρ(α) at each order, which is why you keep a whole vector of orders and only
// » collapse to (ε, δ) at the very end, choosing whichever order gives the
// » smallest ε. That "convert last" discipline is the entire accuracy win.
type RDPAccountant struct {
	mu     sync.Mutex
	orders []float64
	rdp    []float64 // rdp[i] is accumulated ρ at orders[i]
}

// DefaultOrders is the conventional order grid; it covers the useful range of
// noise multipliers without being expensive to carry.
//
// » Why these? Small α is where large-σ mechanisms convert best, large α where
// » small-σ ones do. If your reported ε is minimised at the FIRST or LAST order
// » in the grid, the grid is too narrow and your ε is being overstated — assert
// » on that in the test rather than discovering it in a review.
var DefaultOrders = []float64{
	1.25, 1.5, 1.75, 2, 2.5, 3, 3.5, 4, 4.5, 5, 6, 7, 8, 9, 10, 12, 14, 16,
	20, 24, 28, 32, 48, 64, 128, 256, 512,
}

// NewRDPAccountant returns an accountant over the given orders; nil means
// DefaultOrders.
func NewRDPAccountant(orders []float64) *RDPAccountant {
	if len(orders) == 0 {
		orders = DefaultOrders
	}
	cp := make([]float64, len(orders))
	copy(cp, orders)
	return &RDPAccountant{orders: cp, rdp: make([]float64, len(cp))}
}

// Add implements Accountant.
//
// TODO(week1): implement.
func (a *RDPAccountant) Add(noiseMultiplier, q float64, steps int) error {
	// EXERCISE-BEGIN
	// ─── EXERCISE 6: RDP for the Gaussian mechanism ─────────────────────────
	// Task: for q == 1, accumulate ρ(α) += steps · α / (2·σ²) at every order.
	//       Validate σ > 0, steps > 0, q ∈ (0, 1].
	// Reference: Mironov, "Rényi Differential Privacy" (arXiv 1702.07476), §3
	//       and Table 1 — Proposition 7 is the Gaussian bound above.
	// ────────────────────────────────────────────────────────────────────────

	// ─── EXERCISE 7 (do this one in week 6, not week 1) ─────────────────────
	// Subsampling amplification: when only a q-fraction of contributors take
	// part in a release, privacy is amplified, and the bound is no longer a
	// closed form — it is a log of a binomial-style sum over the order.
	// This is the single ingredient that makes DP-SGD's ε tolerable, so you
	// will need it for Project B. Implement the sampled-Gaussian RDP bound of
	// Mironov, Talwar & Zhang (arXiv 1908.10530) §3, for integer α, and fall
	// back to the q = 1 bound for non-integer orders.
	// Sanity check that matters: at q = 1 the amplified bound must reduce
	// exactly to α/(2σ²). If it does not, you have a sign or index error.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return ErrTODO
}

// Total implements Accountant: converts accumulated RDP to (ε, δ).
//
// TODO(week1): implement.
func (a *RDPAccountant) Total(delta float64) (float64, error) {
	// EXERCISE-BEGIN
	// ─── EXERCISE 8: RDP → (ε, δ) conversion ────────────────────────────────
	// Task: implement the standard conversion and return the minimum over
	//       orders. The textbook version (Mironov Prop. 3) is
	//           ε(α) = ρ(α) + log(1/δ) / (α − 1)
	//       The tighter one used by modern libraries (Canonne–Kamath–Steinke
	//       Prop. 12 / Balle et al.) is
	//           ε(α) = ρ(α) + log((α−1)/α) − (log δ + log α)/(α − 1)
	//       Implement the textbook one first, get the test green, then swap in
	//       the tighter one and record the difference — it is a few percent,
	//       and knowing that it is only a few percent is itself the useful
	//       result.
	// Also return WHICH order minimised ε (add a method or a second return
	//       value). You want that in the logs: it is how you notice the grid is
	//       too narrow in production.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return 0, ErrTODO
}

// ─── the operational half: per-task budget enforcement ───────────────────────

// Ledger enforces a per-task ε budget over a validity period. It is the piece
// that turns DP from a property of one function call into a property of the
// deployment.
//
// » Deliberately in-memory and therefore lost on restart. That is a BUG, not a
// » simplification, and it is the most instructive thing in this file: a
// » restart loop that resets the ledger lets an operator spend unbounded
// » budget without noticing. See« EXERCISE 9». Write it down in
// » THREAT_MODEL.md under "what breaks if the leader restarts".
type Ledger struct {
	mu    sync.Mutex
	spent map[string]float64
	limit map[string]float64
}

// NewLedger returns an empty ledger.
func NewLedger() *Ledger {
	return &Ledger{spent: map[string]float64{}, limit: map[string]float64{}}
}

// SetBudget declares the total ε available to taskID for the current period.
func (l *Ledger) SetBudget(taskID string, epsilon float64) error {
	if epsilon <= 0 {
		return fmt.Errorf("dp: budget for task %q must be > 0", taskID)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.limit[taskID] = epsilon
	return nil
}

// Spend deducts epsilon from taskID's budget, or fails without deducting.
//
// » Atomic check-and-deduct under one lock: a check() followed by a separate
// » spend() is a TOCTOU race that lets two concurrent collections both pass the
// » check and jointly overspend. The privacy guarantee is exactly the kind of
// » invariant that must not have a race in it.
func (l *Ledger) Spend(taskID string, epsilon float64) error {
	if epsilon <= 0 || math.IsNaN(epsilon) {
		return fmt.Errorf("dp: spend for task %q must be a positive number", taskID)
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	limit, ok := l.limit[taskID]
	if !ok {
		return fmt.Errorf("dp: no budget configured for task %q", taskID)
	}
	if l.spent[taskID]+epsilon > limit {
		return fmt.Errorf("dp: task %q budget exhausted: spent %g + %g > limit %g",
			taskID, l.spent[taskID], epsilon, limit)
	}
	l.spent[taskID] += epsilon
	return nil
}

// Remaining reports the unspent budget for taskID.
func (l *Ledger) Remaining(taskID string) float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit[taskID] - l.spent[taskID]
}

// EXERCISE-BEGIN
// ─── EXERCISE 9: make the ledger survive the process ────────────────────────
// The in-memory ledger above is defeated by `kubectl rollout restart`.
//
// Task A: persist it. The cheapest durable option that fits this project's
//         existing shape is a Kafka topic (compacted, key = task ID) — you
//         already run one, and internal/sink already knows how to produce to
//         it. Replay it at startup to rebuild `spent`. Think about whether you
//         need the write to be durable BEFORE you publish the aggregate, and
//         which order is safe (hint: the same reasoning as write-ahead logging
//         — spend first, publish second, because double-spending is a privacy
//         violation and double-publishing the same aggregate is not).
// Task B: add budget REFRESH: budgets are usually per-period ("ε = 4 per
//         epoch"). Add a period to SetBudget and reset spent at each boundary.
//         Then argue in docs/PRIVACY.md why that is sound — the answer involves
//         a user's data appearing in only one period, and it is an assumption
//         about the data, not a theorem. Say so explicitly.
// Task C: export `dp_task_epsilon_spent` / `dp_task_epsilon_limit` gauges via
//         internal/privacy/metrics.go, and alert at 80%. An exhausted budget is
//         a page: it means metrics silently stop updating.
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END
