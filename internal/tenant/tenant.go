// Package tenant implements the admission layer: who is calling (auth) and
// how much they may send (per-tenant rate limiting).
//
// » This is the "Admission" box on the collector hot-path diagram. The two
// » jobs are deliberately fused: identity is a prerequisite for isolation.
// » Background reading on why per-tenant isolation matters in shared
// » ingestion systems: https://aws.amazon.com/builders-library/fairness-in-multi-tenant-systems/
package tenant

import (
	"context"
	"crypto/sha256"
	"strings"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/tiennvhust/gpu-trace-collector/internal/config"
	"github.com/tiennvhust/gpu-trace-collector/internal/obs"
)

// headerAPIKey is the gRPC metadata key carrying the tenant credential.
//
// » The agent needs no code change: the OTel SDK forwards
// » OTEL_EXPORTER_OTLP_HEADERS="x-api-key=..." as gRPC metadata.
// » https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/
const headerAPIKey = "x-api-key"

// Tenant is the runtime representation of one configured tenant.
type Tenant struct {
	Name string
	// Limiter is a token bucket: capacity = burst, refill = events/sec.
	//
	// » Token bucket is the standard choice for ingest admission because it
	// » allows short bursts (agents flush on an interval, so traffic is
	// » naturally bursty) while bounding the long-run average rate.
	// » Chapter: https://sre.google/sre-book/handling-overload/
	Limiter *rate.Limiter
}

// EXERCISE-BEGIN
// ─── EXERCISE 1: harden the credential check ────────────────────────────────
// The map lookup below leaks timing information and supports exactly one key
// per tenant, which makes key rotation a hard cutover.
//
 // Task A: compare keys with crypto/subtle.ConstantTimeCompare. Note you can't
//         constant-time a map lookup directly — think about hashing the
//         presented key first (e.g. sha256) and indexing by the digest.
// Task B: allow each tenant TWO active keys (primary + next) so rotation is
//         zero-downtime: add key2 → move agents → remove key1.
// References:
//   https://pkg.go.dev/crypto/subtle#ConstantTimeCompare
//   https://codahale.com/a-lesson-in-timing-attacks/
// The baseline works without this — it is a hardening exercise.
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END

// Registry maps SHA-256 digests of API keys to tenants. Built once at
// startup, read-only after, therefore safe for concurrent use without locks.
//
// » Indexing by digest instead of the raw key removes the timing side channel
// » of a string-keyed map: lookup time now depends only on the hash, and
// » crafting keys whose digests share a prefix is a preimage attack on
// » SHA-256. A tenant may appear under two digests during key rotation.
type Registry struct {
	byKey map[[32]byte]*Tenant
}

// NewRegistry builds the registry from configuration.
func NewRegistry(tenants []config.Tenant) *Registry {
	r := &Registry{byKey: make(map[[32]byte]*Tenant, len(tenants))}
	for _, t := range tenants {
		key := sha256.Sum256([]byte(t.APIKey))
		tenant := &Tenant{
			Name:    t.Name,
			Limiter: rate.NewLimiter(rate.Limit(t.EventsPerSec), t.Burst),
		}
		r.byKey[key] = tenant

		if t.APIKey2 != "" {
			key2 := sha256.Sum256([]byte(t.APIKey2))
			r.byKey[key2] = tenant
		}
	}
	return r
}

// Lookup resolves an API key to a tenant.
func (r *Registry) Lookup(key string) (*Tenant, bool) {
	t, ok := r.byKey[sha256.Sum256([]byte(key))]
	return t, ok
}

// EXERCISE-BEGIN
// ─── EXERCISE 2: add a collector-wide limiter ───────────────────────────────
// Per-tenant limits protect tenants from each other, but nothing protects the
// PROCESS: if you configure 50 tenants at 10k events/s each, the sum can
// exceed what one collector replica can handle.
//
// Task: add a second, global rate.Limiter (config: global_events_per_sec)
//       checked AFTER the per-tenant one in the OTLP handlers. Reject with
//       the same ResourceExhausted code but reason label "global_rate_limit"
//       so the two rejection causes are distinguishable on the dashboard.
// Think about: why check per-tenant FIRST? (Hint: a global-first check lets
//       one noisy tenant consume the global budget and starve the others —
//       exactly the unfairness this layer exists to prevent.)
// ─────────────────────────────────────────────────────────────────────────────
// EXERCISE-END

type ctxKey struct{}

// FromContext returns the tenant injected by the auth interceptor.
func FromContext(ctx context.Context) *Tenant {
	t, _ := ctx.Value(ctxKey{}).(*Tenant)
	return t
}

// UnaryAuthInterceptor authenticates every unary RPC and injects the resolved
// tenant into the request context.
//
// » Interceptors are gRPC middleware — auth lives here so the OTLP handlers
// » never see an unauthenticated request. https://grpc.io/docs/guides/interceptors/
func UnaryAuthInterceptor(reg *Registry, m *obs.Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (any, error) {

		// » Kubernetes probes and grpcurl health checks must not need an API
		// » key, so the standard health service bypasses auth.
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}

		md, _ := metadata.FromIncomingContext(ctx)
		keys := md.Get(headerAPIKey)
		if len(keys) == 0 {
			m.Rejected.WithLabelValues("unknown", "unauthenticated").Inc()
			return nil, status.Error(codes.Unauthenticated, "missing "+headerAPIKey+" header")
		}
		t, ok := reg.Lookup(keys[0])
		if !ok {
			m.Rejected.WithLabelValues("unknown", "unauthenticated").Inc()
			// » Same message for "no key" and "bad key": error text must not
			// » become an oracle that confirms which keys exist.
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return handler(context.WithValue(ctx, ctxKey{}, t), req)
	}
}
