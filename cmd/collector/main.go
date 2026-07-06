// gpu-trace-collector: multi-tenant OTLP ingestion → Kafka.
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

	kafka, err := sink.New(cfg.Kafka, metrics)
	if err != nil {
		log.Fatalf("sink: %v", err)
	}
	queue := pipeline.New(cfg.QueueCapacity, cfg.Workers, kafka, metrics)
	metrics.RegisterQueueDepth(queue.Depth)

	reg := tenant.NewRegistry(cfg.Tenants)
	gs := grpc.NewServer(
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgBytes),
		grpc.ChainUnaryInterceptor(tenant.UnaryAuthInterceptor(reg, metrics)),
	)
	server.Register(gs, queue, metrics)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(gs, hs)
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

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
