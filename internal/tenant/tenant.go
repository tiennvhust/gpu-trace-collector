// Package tenant implements the admission layer: who is calling (auth) and
// how much they may send (per-tenant rate limiting).
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
const headerAPIKey = "x-api-key"

// Tenant is the runtime representation of one configured tenant.
type Tenant struct {
	Name string
	// Limiter is a token bucket: capacity = burst, refill = events/sec.
	Limiter *rate.Limiter
}

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

type ctxKey struct{}

// FromContext returns the tenant injected by the auth interceptor.
func FromContext(ctx context.Context) *Tenant {
	t, _ := ctx.Value(ctxKey{}).(*Tenant)
	return t
}

// UnaryAuthInterceptor authenticates every unary RPC and injects the resolved
// tenant into the request context.
func UnaryAuthInterceptor(reg *Registry, m *obs.Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (any, error) {

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
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return handler(context.WithValue(ctx, ctxKey{}, t), req)
	}
}
