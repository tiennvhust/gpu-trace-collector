// gpu-trace-collector: multi-tenant OTLP ingestion → Kafka.
//
// » Data path (see README diagram):
// »   agent ──OTLP/gRPC──▶ auth ▶ rate limit ▶ bounded queue ▶ Kafka
// » Everything durable lives in Kafka; everything here is bounded and
// » in-memory — which is what makes this process stateless and therefore
// » trivially horizontally scalable (a plain Deployment + HPA on AKS).
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/tiennvhust/gpu-trace-collector/internal/config"
	"github.com/tiennvhust/gpu-trace-collector/internal/obs"
	"github.com/tiennvhust/gpu-trace-collector/internal/pipeline"
	"github.com/tiennvhust/gpu-trace-collector/internal/server"
	"github.com/tiennvhust/gpu-trace-collector/internal/sink"
	"github.com/tiennvhust/gpu-trace-collector/internal/tenant"
)

func main() {
	cfgPath := flag.String("config", "configs/collector.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	metrics := obs.New()

	// » Construction order mirrors the data path in reverse — each stage
	// » needs its downstream to exist first: sink → queue → gRPC services.
	kafka, err := sink.New(cfg.Kafka, metrics)
	if err != nil {
		log.Fatalf("sink: %v", err)
	}
	queue := pipeline.New(cfg.QueueCapacity, cfg.Workers, kafka, metrics, cfg.OverloadPolicy)
	metrics.RegisterQueueDepth(queue.Depth)

	reg := tenant.NewRegistry(cfg.Tenants)

	// » global_events_per_sec absent/zero means "no global limit": rate.Inf
	// » makes AllowN always true, so the handlers need no nil check.
	lim := rate.NewLimiter(rate.Inf, 0)
	if cfg.GlobalEventsPerSec > 0 {
		lim = rate.NewLimiter(rate.Limit(cfg.GlobalEventsPerSec), cfg.GlobalBurst)
	}

	gs := grpc.NewServer(
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgBytes),
		grpc.ChainUnaryInterceptor(tenant.UnaryAuthInterceptor(reg, metrics)),
	)
	server.Register(gs, queue, metrics, lim)

	// » The standard gRPC health service: what a Kubernetes readinessProbe
	// » (grpc probe type) will call. Readiness should mean "can do useful
	// » work" — we proved Kafka reachability at startup via Ping; a later
	// » refinement is flipping NOT_SERVING when produce errors spike.
	// » https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
	hs := health.NewServer()
	healthpb.RegisterHealthServer(gs, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// EXERCISE-BEGIN
	// ─── EXERCISE 7 (stretch): profiling endpoint ───────────────────────────
	// Add net/http/pprof to the HTTP mux (import _ "net/http/pprof" exposes
	// /debug/pprof on the DefaultServeMux — decide whether that's acceptable
	// or whether you want it on the private mux only). Then, under load from
	// your generator, capture a CPU profile, find the top allocation site on
	// the hot path, fix ONE thing, and record before/after in BENCHMARKS.md.
	// https://go.dev/blog/pprof
	// ─────────────────────────────────────────────────────────────────────────
	// EXERCISE-END

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	httpSrv := &http.Server{Addr: cfg.HTTPListen, Handler: mux}
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", cfg.GRPCListen)
	if err != nil {
		log.Fatalf("listen %s: %v", cfg.GRPCListen, err)
	}
	go func() {
		log.Printf("OTLP/gRPC listening on %s, metrics on %s%s",
			cfg.GRPCListen, cfg.HTTPListen, "/metrics")
		if err := gs.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down: draining in-flight requests and queue")

	// » Shutdown is a pipeline drained front-to-back; this ordering is what
	// » makes `kubectl rollout restart` lossless:
	// »   1. GracefulStop: stop accepting RPCs, wait for in-flight handlers
	// »      (every accepted item is now in the queue).
	// »   2. queue.Close: workers drain remaining items into the sink.
	// »   3. kafka.Close: Flush pushes the producer buffer to brokers.
	// » Kubernetes gives terminationGracePeriodSeconds (default 30s) for all
	// » of this — hence the deadline on the flush.
	hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	gs.GracefulStop()
	queue.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := kafka.Close(ctx); err != nil {
		log.Printf("kafka close: %v", err)
	}
	_ = httpSrv.Shutdown(ctx)
	log.Println("bye")
}
