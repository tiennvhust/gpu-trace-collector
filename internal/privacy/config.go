package privacy

import (
	"errors"
	"fmt"
	"time"
)

// » CONFIG IS A PRIVACY INTERFACE, not a convenience. Every field below can be set
// » to a value that quietly destroys the guarantee — an ε of 100, a minimum batch
// » size of 2, a contribution bound that does not match the circuit. So this file's
// » job is to make the dangerous settings HARD TO EXPRESS and IMPOSSIBLE TO DEPLOY
// » SILENTLY.
// »
// » The habit worth building: for each field, ask "what is the worst value someone
// » could set here in a hurry at 2am, and does the code refuse it?" That question
// » produces better validation than any style guide.

// Config is the privacy block of the collector configuration.
//
// » Lives in this package rather than internal/config so that internal/config can
// » import it without a cycle. Same reason internal/config owns the Kafka and
// » Tenant structs but not their behaviour.
type Config struct {
	// Enabled turns the private path on. Absent means off, so an existing
	// deployment's config file keeps working unchanged.
	Enabled bool `yaml:"enabled"`

	// QueueCapacity bounds the private path's in-memory queue. When full,
	// measurements are dropped and counted; the plaintext path is unaffected.
	QueueCapacity int `yaml:"queue_capacity"`

	// Workers is the number of sharding/upload goroutines. Sharding is CPU-bound
	// and uploading is IO-bound; see internal/privacy/path.go«, EXERCISE 64,» on
	// sizing them.
	Workers int `yaml:"workers"`

	// Tasks are the aggregation tasks this collector feeds.
	Tasks []TaskConfig `yaml:"tasks"`
}

// TaskConfig is one aggregation task's configuration.
type TaskConfig struct {
	// ID is the hex task ID, or empty to derive it from the parameters. Deriving
	// is strongly preferred«, see dap EXERCISE 47,» because it turns a
	// configuration mismatch between the two aggregator operators into a startup
	// error instead of a silent privacy downgrade.
	ID string `yaml:"id"`

	// LeaderURL and HelperURL are the aggregators. They MUST be operated by
	// different organisations for the trust model to mean anything; see the
	// validation note in Validate.
	LeaderURL string `yaml:"leader_url"`
	HelperURL string `yaml:"helper_url"`

	// VDAF is "count", "sum" or "histogram".
	VDAF string `yaml:"vdaf"`

	// Buckets and ChunkLength configure "histogram".
	Buckets     int `yaml:"buckets"`
	ChunkLength int `yaml:"chunk_length"`

	// MaxValue configures "sum": the per-report contribution bound.
	MaxValue uint64 `yaml:"max_value"`

	// Metric selects and shapes the OTLP input.
	Metric MetricConfig `yaml:"metric"`

	// TimePrecision is the batch granularity.
	TimePrecision time.Duration `yaml:"time_precision"`

	// MinBatchSize is the smallest collectable batch.
	MinBatchSize int `yaml:"min_batch_size"`

	// Epsilon and Delta are the per-collection privacy parameters; Budget is the
	// total ε available per period.
	Epsilon float64 `yaml:"epsilon"`
	Delta   float64 `yaml:"delta"`
	Budget  float64 `yaml:"epsilon_budget"`

	// VerifyKey is the aggregators' shared secret, hex-encoded. Comes from the
	// environment via ${...} expansion, never from the file — the same rule
	// internal/config already applies to API keys and SASL passwords.
	VerifyKey string `yaml:"verify_key"`
}

// MetricConfig maps an OTLP metric onto the task's measurement.
type MetricConfig struct {
	Name string `yaml:"name"`

	// Bucketing is "linear" or "log", for histogram tasks.
	Bucketing string  `yaml:"bucketing"`
	Lo        float64 `yaml:"lo"`
	Hi        float64 `yaml:"hi"`

	// Threshold is the value above which a count task records a 1.
	Threshold float64 `yaml:"threshold"`
}

// Default values, chosen conservatively: a default that is wrong should fail
// loudly rather than produce weak privacy.
const (
	DefaultQueueCapacity = 8192
	DefaultWorkers       = 2

	// DefaultMinBatchSize is deliberately large. A small batch is the single
	// easiest way to defeat this whole system, so the default errs toward "no data
	// yet" rather than "data that leaks".
	DefaultMinBatchSize = 1000

	// DefaultTimePrecision matches a plausible fleet reporting cadence.
	DefaultTimePrecision = time.Hour
)

// ApplyDefaults fills in unset fields.
//
// TODO(week4): implement.
func (c *Config) ApplyDefaults() {
	if c.QueueCapacity <= 0 {
		c.QueueCapacity = DefaultQueueCapacity
	}
	if c.Workers <= 0 {
		c.Workers = DefaultWorkers
	}
	// EXERCISE-BEGIN
	// ─── EXERCISE 66: which fields get defaults, and which must not ──────────
	// Task: default TimePrecision and MinBatchSize per task.
	//
	// Then the decision that matters, and think about it rather than copying the
	// pattern above: should Epsilon have a default?
	//
	// The argument for NO: ε is a policy decision with no universally sane value.
	// Defaulting it means a task can be deployed by someone who never thought about
	// the privacy parameter at all, and the config file will not even show that a
	// choice was made. Requiring it forces the conversation, and the conversation is
	// the control.
	//
	// The argument for yes is convenience, which loses.
	//
	// So: require Epsilon, Delta and Budget explicitly, and make Validate reject a
	// task without them. Then note in docs/PRIVACY.md that this was deliberate.
	// "Which settings did you refuse to give a default" is a question that reveals
	// whether someone has thought about a config surface as a safety surface.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
}

// Validate checks the configuration.
//
// TODO(week4): implement.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if len(c.Tasks) == 0 {
		return errors.New("privacy: enabled but no tasks configured")
	}
	// EXERCISE-BEGIN
	// ─── EXERCISE 67: validation that refuses bad privacy ───────────────────
	// Per task, check the mechanical things (URLs non-empty, VDAF recognised,
	// buckets > 0 for histograms, verify key present and the right length) and then
	// the four that are actually about privacy:
	//
	//   1. LEADER AND HELPER MUST DIFFER. Same host means one party holds both
	//      shares and the entire trust model is void — while every test still
	//      passes and every dashboard looks healthy. Refuse it. Then go further:
	//      compare the registrable domains, not just the URLs, and log a warning if
	//      they match, because two subdomains of one company is not two
	//      organisations. You cannot fully enforce non-collusion in code, but you
	//      can refuse the cases that are obviously wrong — and being clear about
	//      which parts of a threat model are enforceable and which are contractual
	//      is exactly the distinction to draw in THREAT_MODEL.md.
	//
	//   2. Budget >= Epsilon, or the task cannot complete a single collection.
	//
	//   3. MinBatchSize sanity, with a loud warning below 1000 and a hard floor
	//      somewhere (100?) below which you refuse outright. Pick the floor and
	//      justify it.
	//
	//   4. EXPECTED UTILITY, the one that makes this feel like real engineering:
	//      compute the expected per-bucket noise from Epsilon and the bucket count
	//      (dp.Histogram.PerBucketStdDev, EXERCISE 5) and compare it against the
	//      expected per-bucket signal (MinBatchSize / Buckets). If the noise
	//      dominates, this task will produce nothing but noise — REFUSE TO START,
	//      with a message that says what to change:
	//
	//        privacy: task gpu-util: at epsilon=0.1 with 1024 buckets and a
	//        minimum batch of 1000, expected per-bucket noise (±14.1) exceeds the
	//        expected per-bucket count (1.0). Increase epsilon, reduce buckets, or
	//        raise min_batch_size.
	//
	//      That error message is a small thing that will read, to anyone who has
	//      operated a system, as though written by someone who has.
	//
	// Return all problems at once with errors.Join.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END
	return ErrTODO
}

// validateTask checks one task's configuration.
//
// TODO(week4): implement« — see EXERCISE 67».
func validateTask(t TaskConfig) error {
	if t.Epsilon <= 0 {
		return fmt.Errorf("privacy: task %q: epsilon must be set explicitly and > 0", t.Metric.Name)
	}
	return ErrTODO
}
