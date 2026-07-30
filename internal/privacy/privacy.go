// Package privacy is the bridge between this collector's plaintext OTLP path and
// the private aggregation path: it turns OTLP datapoints into VDAF measurements
// and ships them as DAP reports.
//
// » THIS PACKAGE IS THE POINT OF PROJECT A. The other three packages
// » (internal/dp, internal/vdaf, internal/dap) are a faithful implementation of
// » published specifications — valuable, but someone else designed them. This one
// » is the design decision that makes the project yours:
// »
// »   The collector already sees every tenant's raw telemetry in the clear. Add a
// »   path where it CANNOT.
// »
// » And the reason to extend this repository rather than start a fresh one: a
// » two-year-old production service that grew a privacy layer reads far better
// » than a greenfield toy. It demonstrates the ideation → production → operation
// » arc, and it keeps the existing eBPF/Go work in the story instead of orphaning
// » it. The CV bullet is "added a privacy-preserving path to a multi-tenant
// » telemetry measurement pipeline I had already shipped", which is a different
// » and much stronger claim than "implemented Prio3".
//
// » THE TWO PATHS, SIDE BY SIDE:
// »
// »   PLAINTEXT (existing)
// »     agent ─OTLP/gRPC→ [auth │ rate limit │ queue] ─→ Kafka ─→ per-device rows
// »     The collector sees everything. Right answer for a tenant's own
// »     infrastructure, where per-device attribution IS the product.
// »
// »   PRIVATE (new)
// »     agent ─OTLP/gRPC→ [auth │ rate limit │ encode │ shard] ─→ DAP leader
// »                                                          ↘ DAP helper
// »                                        collector ←─ noised aggregate only
// »     Nobody sees any device's value. Right answer for fleet-wide statistics,
// »     cross-tenant benchmarks, and anything that leaves the tenant's boundary.
// »
// » BOTH PATHS COEXIST, and that is the honest design rather than a hedge. The
// » interesting engineering question is not "which is better" but "which data
// » belongs on which path, and who decides" — and being able to answer that
// » concretely is worth more in an interview than either implementation.

package privacy

import "errors"

// ErrTODO marks an unimplemented scaffold function.
var ErrTODO = errors.New("privacy: not implemented«, see the EXERCISE block above this function»")

// EXERCISE-BEGIN
// ─── EXERCISE 60 (week 4): decide what belongs on the private path ───────────
// Before writing any more code, write the answer to this in docs/PRIVACY.md,
// because it determines the API and you will otherwise design it twice.
//
// gpu-trace produces, per device: SM utilisation, memory occupancy, kernel
// launch counts, kernel durations, temperature, power draw, CUDA error events,
// process names, PIDs, hostnames.
//
// For each, decide: plaintext, private, or both? And say why. Some hints at the
// shape of a good answer:
//   - process names and hostnames are identifiers, not measurements. They cannot
//     go on the private path at all — there is no VDAF for "collect strings" that
//     is safe here, and Poplar1's heavy hitters is the closest thing (see prio3
//     EXERCISE 21) with real caveats.
//   - kernel durations are a distribution, so Prio3Histogram over log buckets.
//   - CUDA error events are a count: Prio3Count.
//   - utilisation is both — a tenant wants per-device values for its own fleet
//     (plaintext, inside its trust boundary), and the vendor wants the fleet-wide
//     distribution (private, across tenants).
//
// The interesting one to reason about carefully: a single tenant with three GPUs
// is not anonymous in a fleet-wide aggregate no matter how good the crypto is,
// because MinBatchSize counts REPORTS and not TENANTS. Work out what breaks and
// what you would do about it. This is the single best design question in the
// project — and note that the answer is not a cryptographic one.
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END
