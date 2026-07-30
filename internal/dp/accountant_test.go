package dp

import (
	"math"
	"sync"
	"testing"
)

func TestRDPNeverWorseThanBasicComposition(t *testing.T) {
	// » THE test to write first, because it needs no reference values and it
	// » catches sign errors, order-grid mistakes and conversion bugs all at once.
	// » Basic composition is always valid; RDP is a tighter analysis of the same
	// » mechanism. So RDP's ε must be ≤ basic's ε. If it is larger, RDP is wrong
	// » — and note the asymmetry: this test can only ever accuse the accountant
	// » you are writing, never the reference.
	const (
		sigma = 4.0
		steps = 200
		delta = 1e-6
	)

	rdp := NewRDPAccountant(nil)
	if err := rdp.Add(sigma, 1, steps); err != nil {
		t.Fatalf("RDPAccountant.Add: %v", err)
	}
	rdpEps, err := rdp.Total(delta)
	if err != nil {
		t.Fatalf("RDPAccountant.Total: %v", err)
	}

	basic := &BasicAccountant{}
	if err := basic.Add(sigma, 1, steps); err != nil {
		t.Fatalf("BasicAccountant.Add: %v", err)
	}
	basicEps, err := basic.Total(delta * float64(steps))
	if err != nil {
		t.Fatalf("BasicAccountant.Total: %v", err)
	}

	if rdpEps > basicEps {
		t.Errorf("RDP eps %v exceeds basic composition eps %v — the tighter analysis cannot be worse",
			rdpEps, basicEps)
	}
	if rdpEps <= 0 || math.IsInf(rdpEps, 0) || math.IsNaN(rdpEps) {
		t.Errorf("RDP eps = %v, want a finite positive number", rdpEps)
	}
}

func TestRDPScalesSublinearlyInSteps(t *testing.T) {
	// » The whole point of RDP: 100× the releases should cost far less than 100×
	// » the ε. Roughly √k for the Gaussian mechanism, so 100 steps should land
	// » near 10× one step, not 100×. A loose bound (< 30×) still catches an
	// » accountant that has accidentally reverted to linear composition.
	one := NewRDPAccountant(nil)
	if err := one.Add(4, 1, 1); err != nil {
		t.Fatalf("Add: %v", err)
	}
	many := NewRDPAccountant(nil)
	if err := many.Add(4, 1, 100); err != nil {
		t.Fatalf("Add: %v", err)
	}

	e1, err := one.Total(1e-6)
	if err != nil {
		t.Fatalf("Total: %v", err)
	}
	e100, err := many.Total(1e-6)
	if err != nil {
		t.Fatalf("Total: %v", err)
	}
	if e100 >= 30*e1 {
		t.Errorf("eps(100 steps) = %v, eps(1 step) = %v: ratio %v looks linear, not √k",
			e100, e1, e100/e1)
	}
}

func TestRDPMinimisingOrderIsInsideTheGrid(t *testing.T) {
	// » If the best order is at either end of DefaultOrders, the grid does not
	// » cover this mechanism and the reported ε is an overestimate. Once
	// » RDPAccountant reports which order won «(EXERCISE 8)», assert here that it
	// » is neither the first nor the last. Until then this test documents the
	// » requirement.
	t.Skip("enable once RDPAccountant.Total reports the minimising order «(EXERCISE 8)»")
}

func TestLedgerRefusesToOverspend(t *testing.T) {
	l := NewLedger()
	if err := l.SetBudget("task-a", 1.0); err != nil {
		t.Fatalf("SetBudget: %v", err)
	}
	if err := l.Spend("task-a", 0.6); err != nil {
		t.Fatalf("first spend: %v", err)
	}
	if err := l.Spend("task-a", 0.6); err == nil {
		t.Fatal("second spend should exceed the budget, got nil error")
	}
	if got := l.Remaining("task-a"); math.Abs(got-0.4) > 1e-12 {
		t.Errorf("Remaining = %v, want 0.4 (a rejected spend must not deduct)", got)
	}
}

func TestLedgerUnknownTaskIsRejected(t *testing.T) {
	// » Fail closed: an unconfigured task must not get free budget. The
	// » alternative — treating "no entry" as "unlimited" — is how privacy
	// » controls quietly stop applying after a config typo.
	if err := NewLedger().Spend("nope", 0.1); err == nil {
		t.Fatal("spending against an unconfigured task should fail")
	}
}

func TestLedgerSpendIsAtomicUnderConcurrency(t *testing.T) {
	// » The TOCTOU test. 100 goroutines each try to spend 0.1 from a budget of
	// » 1.0; exactly 10 must succeed. A check-then-spend implementation lets
	// » more through and this test catches it, usually only with -race and
	// » -count=20 — run it that way in CI.
	l := NewLedger()
	if err := l.SetBudget("t", 1.0); err != nil {
		t.Fatalf("SetBudget: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := l.Spend("t", 0.1); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if ok != 10 {
		t.Errorf("%d spends of 0.1 succeeded against a budget of 1.0, want 10", ok)
	}
}
