// prio-client: a DAP client and load generator.
//
// » Two jobs in one binary, and both matter:
// »
// »   1. A working client, so you can drive the aggregators end to end without the
// »      collector in the loop. Build this in week 3 alongside the aggregators; it
// »      is how you get the whole private pipeline running before integrating.
// »   2. A LOAD GENERATOR, which is how you produce the numbers in
// »      docs/BENCHMARKS.md. The plan calls for "reports/sec, client-side CPU and
// »      bytes-on-wire overhead vs. the plaintext OTLP path" — this is the tool
// »      that measures it.
// »
// » Make -n large enough to fill a batch (MinBatchSize defaults to 1000) or your
// » collections will keep failing the minimum-batch-size check and you will spend
// » twenty minutes confused. Ask how I know.
package main

import (
	"flag"
	"log"
)

func main() {
	var (
		leaderURL = flag.String("leader", "http://localhost:8080", "DAP leader base URL")
		taskID    = flag.String("task", "", "hex task ID")
		n         = flag.Int("n", 1000, "number of reports to upload")
		rateLimit = flag.Float64("rate", 100, "uploads per second")
		mode      = flag.String("mode", "count", "measurement: count | sum | histogram")
		collect   = flag.Bool("collect", false, "after uploading, run a collection and print the result")
	)
	flag.Parse()

	// EXERCISE-BEGIN
	// ─── EXERCISE 70: the client and load generator ─────────────────────────
	//   1. dap.NewClient against -leader, fetching HPKE configs.
	//   2. Upload -n reports at -rate, with a golang.org/x/time/rate limiter — the
	//      same limiter package the collector already uses for tenant quotas.
	//   3. Report, at the end: uploads/sec achieved, p50/p99 upload latency, total
	//      bytes sent, and bytes per report.
	//   4. With -collect, run a collection and print the noised aggregate NEXT TO
	//      the true value you generated. Seeing "true 500, published 503" on your
	//      terminal is the moment the whole project clicks, and it is also the
	//      single best thing to screenshot for a blog post.
	//
	// THE COMPARISON THAT MATTERS. Add a -plaintext mode that sends the equivalent
	// data as OTLP to the existing collector, so one command produces both halves of
	// the privacy-tax table:
	//
	//   $ prio-client -mode histogram -n 10000 -plaintext
	//     10000 datapoints, 412 KiB, 0.9s, 11,100/s
	//   $ prio-client -mode histogram -n 10000
	//     10000 reports, 8.4 MiB, 14.2s, 704/s
	//
	// Then you can say "the private path costs 20× the bytes and 15× the wall time
	// at 1024 buckets" and have the command that produced it. That is worth more
	// than any amount of description — and note that the interesting follow-up is
	// which of those two numbers actually matters on a battery, and why (see
	// internal/dap/client.go).
	//
	// Also add -profile to write a CPU profile, so you can answer "where does the
	// client's time go" with a flame graph rather than a guess. The answer will be
	// the XOF and the field multiplications, which is why EXERCISE 11 Task B exists.
	// ────────────────────────────────────────────────────────────────────────
	// EXERCISE-END

	log.Fatalf("EXERCISE 70: implement the client (leader=%s task=%s n=%d rate=%v mode=%s collect=%v)",
		*leaderURL, *taskID, *n, *rateLimit, *mode, *collect)
}
