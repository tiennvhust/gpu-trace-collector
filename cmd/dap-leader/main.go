// dap-leader: the DAP leader aggregator.
//
// » Run this alongside cmd/dap-helper to have a complete two-aggregator
// » deployment on your laptop. Two separate binaries and two separate processes,
// » deliberately — the whole trust model rests on the aggregators being
// » independently operated, so sharing a process, a config file or a secret store
// » between them would be modelling the thing you are trying to avoid.
// »
// » Structurally this mirrors cmd/collector: load config, build the layers
// » bottom-up, serve, drain on SIGTERM. Reuse that file's shutdown ordering
// » reasoning — it is the same problem and the same answer.
package main

import (
	"flag"
	"log"
)

func main() {
	cfgPath := flag.String("config", "configs/dap-leader.yaml", "path to config file")
	flag.Parse()

	// EXERCISE-BEGIN
	// ─── EXERCISE 68: the leader binary ─────────────────────────────────────
	// Wire it up, following cmd/collector/main.go closely:
	//
	//   1. Load config; load HPKE keypairs from the environment (never the file).
	//   2. Build the TaskSet, call Validate on every task, and FAIL TO START on any
	//      problem. A task with bad privacy parameters must not run in a degraded
	//      mode — there is no safe degraded mode for a privacy parameter.
	//   3. dap.NewMetrics(), dap.NewLeader(...).
	//   4. HTTP server on the DAP port with leader.Routes(); a SEPARATE server on
	//      the metrics port. Two listeners, because /metrics must not be reachable
	//      from the public internet — it exposes your task list, your batch fill and
	//      your ε ledger. See the note at the top of internal/dap/metrics.go: the
	//      monitoring system is an actor in the threat model.
	//   5. go leader.RunAggregationJobs(ctx) — the driver.
	//   6. Graceful shutdown, in this order and for these reasons:
	//        a. stop accepting uploads (http.Server.Shutdown on the DAP listener);
	//        b. let in-flight aggregation jobs finish — a job abandoned mid-flight
	//           leaves the helper having aggregated reports the leader did not,
	//           which fails the checksum at collection time and loses the whole
	//           batch;
	//        c. checkpoint the aggregate share if you did EXERCISE 45 Task C;
	//        d. stop the metrics server last, so you can still scrape during (a)–(c).
	//
	// Then: a /healthz that reports NOT_SERVING when the helper has been
	//   unreachable for longer than some threshold. Readiness should mean "can do
	//   useful work", and a leader that cannot reach its helper cannot — the same
	//   reasoning as the health-check comment in cmd/collector/main.go.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END

	log.Fatalf("EXERCISE 68: implement the leader binary (config %s)", *cfgPath)
}
