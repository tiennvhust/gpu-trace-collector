package privacy

import (
	"math"
	"testing"
	"time"
)

func TestLinearBucketEdges(t *testing.T) {
	b := LinearBuckets{Lo: 0, Hi: 100, N: 10}
	cases := []struct {
		v    float64
		want int
	}{
		{0, 0},
		{9.99, 0},
		{10, 1},
		{50, 5},
		{99.99, 9},
		// » Hi itself must land in the LAST bucket, not one past the end. An index
		// » of N is an out-of-range write into the one-hot vector.
		{100, 9},
		// » Out of range clamps into the end buckets rather than being dropped: a
		// » device whose report is missing tells you its value was out of range.
		{-1000, 0},
		{1e9, 9},
	}
	for _, tc := range cases {
		got, ok := b.Bucket(tc.v)
		if !ok {
			t.Errorf("Bucket(%v) returned not-ok", tc.v)
			continue
		}
		if got != tc.want {
			t.Errorf("Bucket(%v) = %d, want %d", tc.v, got, tc.want)
		}
	}
}

func TestLinearBucketRejectsNaNAndInf(t *testing.T) {
	// » NaN comparisons are all false, so a clamp written as two ifs lets NaN
	// » through to int(NaN), which is implementation-defined garbage. GPU telemetry
	// » produces NaN more often than you would like.
	b := LinearBuckets{Lo: 0, Hi: 100, N: 10}
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if i, ok := b.Bucket(v); ok {
			t.Errorf("Bucket(%v) = %d, ok — want rejected", v, i)
		}
	}
}

func TestLinearBucketNeverReturnsOutOfRange(t *testing.T) {
	// » A property test over the whole real line, because the one-hot encoder will
	// » index a slice with whatever this returns.
	b := LinearBuckets{Lo: -50, Hi: 50, N: 7}
	for v := -1000.0; v <= 1000.0; v += 0.37 {
		i, ok := b.Bucket(v)
		if !ok {
			t.Fatalf("Bucket(%v) not ok", v)
		}
		if i < 0 || i >= b.Buckets() {
			t.Fatalf("Bucket(%v) = %d, outside [0, %d)", v, i, b.Buckets())
		}
	}
}

func TestLogBucketsSpreadTheLowEnd(t *testing.T) {
	// » The reason log buckets exist: kernel durations from 2 µs to 200 ms put
	// » ~99% of their mass in linear bucket 0. With log buckets the low end gets
	// » real resolution. This test asserts the property, not specific indices —
	// » 2 µs and 20 µs must land in different buckets, which linear bucketing over
	// » [0, 200ms] with 16 buckets cannot do.
	lb := LogBuckets{Lo: 1e-6, Hi: 0.2, N: 16}
	a, okA := lb.Bucket(2e-6)
	c, okC := lb.Bucket(20e-6)
	if !okA || !okC {
		t.Fatal("LogBuckets rejected in-range values")
	}
	if a == c {
		t.Errorf("2µs and 20µs both landed in bucket %d: no resolution at the low end", a)
	}
}

func TestLogBucketsRejectNonPositiveLo(t *testing.T) {
	// » log(0) is −Inf. A config with lo: 0 must fail at startup, not per datapoint.
	if _, ok := (LogBuckets{Lo: 0, Hi: 1, N: 8}).Bucket(0.5); ok {
		t.Error("LogBuckets with Lo=0 should be rejected")
	}
}

func TestValidateRejectsSameLeaderAndHelper(t *testing.T) {
	// » THE MOST IMPORTANT VALIDATION TEST IN THE PROJECT. Same host for both
	// » aggregators means one party holds both shares and the entire trust model is
	// » void — with every other test still passing and every dashboard healthy.
	cfg := Config{
		Enabled: true,
		Tasks: []TaskConfig{{
			LeaderURL:     "https://dap.example.com",
			HelperURL:     "https://dap.example.com",
			VDAF:          "count",
			Metric:        MetricConfig{Name: "gpu.errors"},
			TimePrecision: time.Hour,
			MinBatchSize:  1000,
			Epsilon:       1,
			Budget:        4,
			VerifyKey:     "00000000000000000000000000000000",
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("a task with the same leader and helper URL must be refused")
	}
}

func TestValidateRejectsEpsilonWithoutBudget(t *testing.T) {
	cfg := Config{
		Enabled: true,
		Tasks: []TaskConfig{{
			LeaderURL:     "https://leader.example.com",
			HelperURL:     "https://helper.example.org",
			VDAF:          "count",
			Metric:        MetricConfig{Name: "gpu.errors"},
			TimePrecision: time.Hour,
			MinBatchSize:  1000,
			Epsilon:       2,
			Budget:        1, // less than one collection's worth
			VerifyKey:     "00000000000000000000000000000000",
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("a budget smaller than epsilon cannot fund a single collection and must be refused")
	}
}

func TestValidateRejectsUnusableUtility(t *testing.T) {
	// »« EXERCISE 67» rule 4: a task whose expected noise dominates its expected
	// » signal produces nothing but noise, and must not start. At ε = 0.1 with 1024
	// » buckets and a minimum batch of 1000, each bucket has an expected count of 1
	// » and a noise standard deviation around 14.
	cfg := Config{
		Enabled: true,
		Tasks: []TaskConfig{{
			LeaderURL:     "https://leader.example.com",
			HelperURL:     "https://helper.example.org",
			VDAF:          "histogram",
			Buckets:       1024,
			ChunkLength:   32,
			Metric:        MetricConfig{Name: "gpu.util", Bucketing: "linear", Lo: 0, Hi: 100},
			TimePrecision: time.Hour,
			MinBatchSize:  1000,
			Epsilon:       0.1,
			Budget:        1,
			VerifyKey:     "00000000000000000000000000000000",
		}},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("a task whose expected noise exceeds its expected signal must be refused at startup")
	}
}

func TestDisabledConfigValidates(t *testing.T) {
	// » An existing deployment's config file, with no privacy block at all, must
	// » keep working unchanged. This test passes out of the box and is here to stop
	// » a future validation change from breaking every existing config.
	if err := (&Config{}).Validate(); err != nil {
		t.Errorf("a disabled privacy config should validate: %v", err)
	}
}

func TestNilPathIsANoOp(t *testing.T) {
	// » privacy.New returns nil when disabled, so the call site in
	// » internal/server/otlp.go needs no branch. That only works if every method is
	// » nil-safe. This test passes now; keep it passing as you add methods.
	var p *Path
	p.Ingest(nil)
	if p.Depth() != 0 {
		t.Error("nil Path should report zero depth")
	}
	p.Close()
}
